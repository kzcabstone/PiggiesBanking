package main

import (
	"github.com/gorilla/mux"
	"encoding/json"
	"net/http"
	"log"
	"strconv"
)

////////////////////////////////////////////////////////////////////////////////
type open_account_request struct {
	Aid 	string 		`json:"aid"`
}

type open_account_response struct {
	Status  string     	`json:"status"`
}

func OpenAccount(w http.ResponseWriter, r *http.Request) interface{} {
	jsondecoder := json.NewDecoder(r.Body)
	var i open_account_request
	u := new(open_account_response)
	if err := jsondecoder.Decode(&i); err != nil {
		u.Status = "Error"
		dumpHttpRequest(r);
		return u
	}

	var result_channel chan string = make(chan string, 2) // small size just for receiving status
	cmd := account_mutator_cmd{Data: account_mutator_cmd_data{Cmdtype: 1, Aid: i.Aid}, Result_channel: result_channel}
	account_mutator_channel <- cmd

	status := <- result_channel
	if status == "OK" {
		u.Status = "OK"
	} else {
		u.Status = "OpenAccount: Server Internal Error " + status
	}
	log.Printf("OpenAccount: id %s, %s", i.Aid, status)
	return u
}

type deposit_request struct {
	Aid 	string 		`json:"aid"`
	Amount  string		`json:"amount"`
}

type deposit_response struct {
	Status  string     	`json:"status"`
}

func Deposit(w http.ResponseWriter, r *http.Request) interface{} {
	jsondecoder := json.NewDecoder(r.Body)
	var i deposit_request
	u := new(deposit_response)
	if err := jsondecoder.Decode(&i); err != nil {
		u.Status = "Error"
		dumpHttpRequest(r);
		return u
	}

	var result_channel chan string = make(chan string, 2) // small size just for receiving status
	amount,err := strconv.ParseInt(i.Amount, 10, 64)
	if err != nil {
		u.Status = "Deposit: invalid amount " + i.Amount
		log.Printf("Deposit: id %s, amount %s", i.Aid, i.Amount)
	} else {
		cmd := account_mutator_cmd{Data: account_mutator_cmd_data{Cmdtype: 2, Aid: i.Aid, Amount: amount}, Result_channel: result_channel}
		account_mutator_channel <- cmd

		status := <- result_channel
		if status == "OK" {
			u.Status = "OK"
		} else {
			u.Status = "Deposit: Server Internal Error " + status
		}
		log.Printf("Deposit: id %s, amount %s, %s", i.Aid, i.Amount, status)
	}

	return u
}

type transfer_request struct {
	Aid 	string 		`json:"aid"`
	Amount  string		`json:"amount"`
	Aid0	string		`json:"aid0"`
}

type transfer_response struct {
	Status  string     	`json:"status"`
}

func Transfer(w http.ResponseWriter, r *http.Request) interface{} {
	jsondecoder := json.NewDecoder(r.Body)
	var i transfer_request
	u := new(transfer_response)
	if err := jsondecoder.Decode(&i); err != nil {
		u.Status = "Error"
		dumpHttpRequest(r);
		return u
	}

	var result_channel chan string = make(chan string, 2) // small size just for receiving status
	amount, err := strconv.ParseInt(i.Amount, 10, 64)
	if err != nil {
		u.Status = "Transfer: invalid amount " + i.Amount
	} else {
		cmd := account_mutator_cmd{Data: account_mutator_cmd_data{Cmdtype: 3, Aid: i.Aid, Amount: amount, Aid0: i.Aid0}, Result_channel: result_channel}
		account_mutator_channel <- cmd

		status := <- result_channel
		if status == "OK" {
			u.Status = "OK"
		} else {
			u.Status = "Transfer: Server Internal Error " + status
		}
		log.Printf("Transfer: id %s, amount %s, to id %s, %s", i.Aid, i.Amount, i.Aid0, status)
	}
	return u
}


////////////////////////////////////////////////////////////////////////////////
type get_account_request struct {
	Aid 	string 	`json:"aid"`
}

type get_account_response struct {
	AccountStr 	string	`json:"accountstr"`
}

func GetAccountInfo(w http.ResponseWriter, r *http.Request) interface{} {
	vars := mux.Vars(r)
	if vars["aid"] == "" {
		log.Printf("GetAccountInfo: invalid request, no aid. Ignore")
		return nil
	}
	var aid = vars["aid"]

	u := new(get_account_response)
	u.AccountStr = aid

	var result_channel chan string = make(chan string, 2)
	cmd := account_mutator_cmd{Data: account_mutator_cmd_data{Cmdtype: 4, Aid: aid}, Result_channel: result_channel}
	account_mutator_channel <- cmd

	tmp := <- result_channel
	_, err := strconv.Atoi(tmp) 
	if err != nil {
		log.Printf("GetAccountInfo: Server Internal Error. %s", tmp)
	} else {
		u.AccountStr += (":" + tmp)
	}
	
	log.Printf("GetAccountInfo: Account %s", u.AccountStr)
	return u
}

////////////////////////////////////////////////////////////////////////////////
type getallaccounts_request struct {
	Suid 	string		`json:"suid"`
}

type getallaccounts_response struct {
	Accounts 	[]string 	`json:"accounts"`	
}

func GetAllAccounts(w http.ResponseWriter, r *http.Request) interface{} {
	vars := mux.Vars(r)
	if vars["suid"] == "" {
		log.Printf("GetAllAccounts: invalid request, no suid. Ignore")
		return nil
	}

	suid := vars["suid"]
	if !checkSUAuth(suid) {
		log.Printf("GetAllAccounts: auth failed %s", suid)
		return nil
	}

	u := new(getallaccounts_response)

	var result_channel chan string = make(chan string, conf.CmdChanDepth)
	cmd := account_mutator_cmd{Data: account_mutator_cmd_data{Cmdtype: 5}, Result_channel: result_channel}
	account_mutator_channel <- cmd

	tmp := <- result_channel
	cnt, err := strconv.Atoi(tmp) 
	if err != nil {
		u.Accounts = make([]string, 1)
		u.Accounts[0] = "Server Internal Error." + tmp
		log.Printf("GetAllAccounts: Server Internal Error. %s", tmp)
	} else {
		u.Accounts = make([]string, cnt)
		for i:=0; i < cnt; i++ {
			astr := <- result_channel
			u.Accounts[i] = astr
		}
	}

	log.Printf("GetAllAccounts: # of accounts %d", len(u.Accounts))
	return u
}

////////////////////////////////////////////////////////////////////////////////
type save_state_request struct {
	Suid	string		`json:"suid"`
}

type save_state_response struct {
	State  string     	`json:"state"`	
}

func SaveServerState(w http.ResponseWriter, r *http.Request) interface{} {
	jsondecoder := json.NewDecoder(r.Body)
	var i save_state_request
	u := new(save_state_response)
	if err := jsondecoder.Decode(&i); err != nil {
		u.State = "Error"
		dumpHttpRequest(r)
		return u
	}

	if !checkSUAuth(i.Suid) {
		log.Printf("SaveServerState: auth failed %s", i.Suid)
		return nil
	}

	if !serialize() {
		u.State = "error"
	} else {
		u.State = "OK"
	}
	return u
}
