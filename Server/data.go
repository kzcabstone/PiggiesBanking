package main

import (
	"log"
	"strconv"
	"encoding/json"
	"io/ioutil"
)

type account struct {
	Id 		string 		`json:"id"`
	Amount  int64		`json:"amount"`
}

type account_mutator_cmd_data struct {
	Cmdtype 	int 	`json:"cmdtype"` // 1: open account   2: deposit  3: transfer
	Aid 		string 	`json:"aid"`
	Amount 		int64 	`json:"amount"` // Must be valid for Cmdtype 2 and 3
	Aid0 		string 	`json:"aid0"` // must be valid for transfer
}

type account_mutator_cmd struct {
	Data  	account_mutator_cmd_data
	Result_channel chan<- string 	`json:"-"` // this channel(should be buffered) is passed in by user to us, we use it to send back result(feeds of the user in specific, one by one)
}

type commitlog_mutator_cmd struct {
	Cmdtype int // 0: save passed in account_mutator_cmd_data to commit_log 1: serialize commit log 
	Acmd  account_mutator_cmd_data
	Result_channel chan<- string // this is used by the mutator to send back "ok" or the marshalled json string of commit_log
}

var commit_log []account_mutator_cmd_data // This is a "commit log" of all commands the mutator received. This gets serializesed for restoring state
var all_accounts map[string]account
var account_mutator_channel chan account_mutator_cmd  // Used by account mutator to receive all commandsd 
var account_mutator_control_channel chan bool
var commitlog_mutator_channel chan commitlog_mutator_cmd // Use a go routine to maintain commit_log and to dump it to file(ideally this should go to a DB) on every update (as much as possible obviously)
var commitlog_mutator_control_channel chan bool 

// Initializes the maps and the channels
func initialize() {
	all_accounts = make(map[string]account)
	commit_log = make([]account_mutator_cmd_data, 0) // size 0 initially 
	account_mutator_channel = make(chan account_mutator_cmd, conf.CmdChanDepth) // the channel only blocks if it's full
	commitlog_mutator_channel = make(chan commitlog_mutator_cmd, conf.CmdChanDepth)
	account_mutator_control_channel = make(chan bool)
	commitlog_mutator_control_channel = make(chan bool)
	// Start the mutator
	go account_mutator()
	go commitlog_mutator()
	if deserialize() {
		replayCommitLog()
	}

}

func uninitialize() {
	if !serialize() {
		log.Printf("uninitialize(): failed to dump state to file %s", conf.Datafile)
		return
	}
	log.Printf("uninitialize(): dumped state to file %s", conf.Datafile)
}

func serialize() bool {
	var result_channel chan string = make(chan string, 2) // only 1 string expected to come out of this
	cmd := commitlog_mutator_cmd{Cmdtype: 1, Result_channel: result_channel}
	commitlog_mutator_channel <- cmd
	str := <- result_channel
	err := ioutil.WriteFile(conf.Datafile, []byte(str), 0644)
	if err != nil {
		log.Printf("serialize(): writing to file %s failed", conf.Datafile)
		return false
	}
	log.Printf(str)
	return true
}

func deserialize() bool {
	dat, err := ioutil.ReadFile(conf.Datafile)
    if err != nil {
    	log.Printf("deserialize(): reading from file %s failed", conf.Datafile)
    	return false
    } else {
    	if e := json.Unmarshal(dat, &commit_log); e != nil {
    		log.Printf("deserialize(): failed to unmarshal to commit_log")
    		return false
    	}
	    log.Printf(string(dat))
	    log.Printf("deserialize(): success. commit_log.size: %d", len(commit_log))
	    return true
	}
}

func resultWatcher(thecmd account_mutator_cmd_data, rchan chan string, label string) {			
	result := <-rchan
	if result != "OK" {
		t, e := json.Marshal(thecmd)
		if e == nil {
			log.Printf("%s failed. %s, cmd: %s", label, result, t)
		} else {
			log.Printf("%s failed. %s", label, result)
		}
	}
}

func replayCommitLog() {
	for _, cmd := range commit_log {
		var result_channel chan string = make(chan string, 2)
		acmd := account_mutator_cmd{Data: cmd, Result_channel: result_channel}
		account_mutator_channel <- acmd
		go resultWatcher(cmd, result_channel, "replayCommitLog")
	}
}

// handles read/write to all_accounts
// TODO: use https://golang.org/src/sync/rwmutex.go to separate readers and writers 
// 			then we can have mutators to truely only serve mutatable requests
//          and let the reader to deal with all readonly request
func account_mutator() {
	for {
		select {
			case cmd := <-account_mutator_channel:
				processAccountCommand(cmd)
			case stop := <-account_mutator_control_channel:
				if stop {
					log.Printf("Received stop signal. Exiting account_mutator")
					return
				}
		}
	}
}

func setAccountAmount(id string, amount int64) {
		all_accounts[id] = account{Id: id, Amount: amount}
}

func hasSufficientFund(id string, delta int64) bool {
	return all_accounts[id].Amount + delta >= 0
}

func sendCmdToCommitlog(cmd account_mutator_cmd_data) {
	result_channel := make(chan string, 2)
	ccmd := commitlog_mutator_cmd{Cmdtype: 0, Acmd: cmd, Result_channel: result_channel}
	commitlog_mutator_channel <- ccmd
	go resultWatcher(cmd, result_channel, "sendCmdToCommitlog")
}

func processAccountCommand(accountcmd account_mutator_cmd) {
	cmd := accountcmd.Data

	_, exists := all_accounts[cmd.Aid]
	// 1: open account   2: deposit  3: transfer  4: get one account info   5: get all accounts
	if cmd.Cmdtype == 1 {
		if exists {
			log.Printf("processAccountCommand(): account already exists. %s", cmd.Aid)
			return
		} 

		sendCmdToCommitlog(cmd)
		nptr := new(account)
		nptr.Id = cmd.Aid
		nptr.Amount = 0
		all_accounts[cmd.Aid] = *nptr
		accountcmd.Result_channel <- "OK"
		log.Printf("processAccountCommand(): opened account %s", cmd.Aid)
	} else if cmd.Cmdtype == 2 {
		if !exists {
			// do nothing, send back error
			accountcmd.Result_channel <- "Account not found"
			log.Printf("processAccountCommand(): account %s doesnot exist. ignore", cmd.Aid)
			return
		}
		if !hasSufficientFund(cmd.Aid, cmd.Amount) {
			accountcmd.Result_channel <- "Insufficient fund"
			log.Printf("processAccountCommand(): account %s doesnot have sufficient fund %d, %d. ignore", cmd.Aid, all_accounts[cmd.Aid].Amount, cmd.Amount)
			return
		}
		sendCmdToCommitlog(cmd)
		setAccountAmount(cmd.Aid, cmd.Amount + all_accounts[cmd.Aid].Amount)
		accountcmd.Result_channel <- "OK"
		log.Printf("processAccountCommand(): depositted %d to account %s, new balance %d", cmd.Amount, cmd.Aid, all_accounts[cmd.Aid].Amount)
	} else if cmd.Cmdtype == 3 {
		_, destination_account_exists := all_accounts[cmd.Aid0]
		if !exists || !destination_account_exists {
			accountcmd.Result_channel <- "Account not found"
			log.Printf("processAccountCommand(): account %s or %s doesnot exist. ignore", cmd.Aid, cmd.Aid0)
			return
		}
		if !hasSufficientFund(cmd.Aid, -1 * cmd.Amount) {
			accountcmd.Result_channel <- "Insufficient fund"
			log.Printf("processAccountCommand(): account %s doesnot have sufficient fund %d, %d. ignore", cmd.Aid, all_accounts[cmd.Aid].Amount, -1 * cmd.Amount)
			return
		}
		if !hasSufficientFund(cmd.Aid0, cmd.Amount) {
			accountcmd.Result_channel <- "Insufficient fund"
			log.Printf("processAccountCommand(): account %s doesnot have sufficient fund %d, %d. ignore", cmd.Aid0, all_accounts[cmd.Aid0].Amount, cmd.Amount)
			return
		}
		sendCmdToCommitlog(cmd)
		setAccountAmount(cmd.Aid, all_accounts[cmd.Aid].Amount - cmd.Amount)
		setAccountAmount(cmd.Aid0, all_accounts[cmd.Aid0].Amount + cmd.Amount)
		accountcmd.Result_channel <- "OK"
		log.Printf("processAccountCommand(): transfered %d from account %s(%d) to account %s(%d)", 
			cmd.Amount, cmd.Aid, all_accounts[cmd.Aid].Amount, cmd.Aid0, all_accounts[cmd.Aid0].Amount)
	} else if cmd.Cmdtype == 4 {
		if !exists {
			accountcmd.Result_channel <- "Account not found"
			log.Printf("processAccountCommand(): account %s doesnot exist. ignore", cmd.Aid)
			return
		}
		accountcmd.Result_channel <- strconv.FormatInt(all_accounts[cmd.Aid].Amount, 10)
		log.Printf("processAccountCommand(): getAccountInfo %s, %d", cmd.Aid, all_accounts[cmd.Aid].Amount)
		return
	} else if cmd.Cmdtype == 5 {
		accountcmd.Result_channel <- strconv.Itoa(len(all_accounts))
		for _, account := range all_accounts {
			str := account.Id + ":" + strconv.FormatInt(account.Amount, 10)
			accountcmd.Result_channel <- str
		}
		log.Printf("processAccountCommand(): getAllAccounts %d", len(all_accounts))
	} else {
		log.Printf("processAccountCommand(): unsupported cmd %d", cmd.Cmdtype)
		accountcmd.Result_channel <- "Unsupported cmd"
	}
}

// handles read/write/serialize to commit_log
func commitlog_mutator() {
	for {
		select {
			case cmd := <-commitlog_mutator_channel:
				processCommitlogCommand(cmd)
			case stop := <-commitlog_mutator_control_channel:
				if stop {
					log.Printf("Received stop signal. Exiting commitlog_mutator")
					return
				}
		}
	}
}

func saveCommitlog() ([]byte, error) {
	t, e := json.Marshal(commit_log)
	return t, e
}

func processCommitlogCommand(cmd commitlog_mutator_cmd) {
	if cmd.Cmdtype == 0 {		
		// Only do one thing here: save the account_mutator_cmd to the commit_log
		commit_log = append(commit_log, cmd.Acmd)
		cmd.Result_channel <- "OK"
	} else if cmd.Cmdtype == 1 {
		// We send back the json string to avoid slow file IO in the mutator
		t, e := saveCommitlog()
		if e != nil {
			cmd.Result_channel <- "ERROR"
			log.Printf("processCommitlogCommand(): error marshaling commit_log")
		} else {
			cmd.Result_channel <- string(t)
			log.Printf("processCommitlogCommand(): saved commit_log to json.")
		}
	}
}
