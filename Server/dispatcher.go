package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)


/*

1. open an account
/account/   POST  {account_number: "123456"}
2. deposite to an account
/deposit/   POST  {account_number: "123456", amount: "30000"}
3. withdraw from an account
/deposit/   POST  {account_number: "123456", amount: "-30000"}
4. query account value
/account/123456  GET
5. transfer between 2 accounts
/transfer/  POST  {from: "123456", to: "654321", amount: "30000"}
6. Superuser query all account details
/su/accounts   GET   

*/
type config struct {
	HttpPort int `json:"http_port"`
	CmdChanDepth int `json:"cmd_channel_depth"`
	ResultChanDepth int `json:"result_channel_depth"`
	Suid 	string `json:"su_id"`
	Datafile string `json:"datafile"`
}
var conf config

func commonWrapper(f func(http.ResponseWriter, *http.Request) interface{}) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		u := f(w, r)
		b, _ := json.Marshal(u)
		w.Write(b)
	}
}

func main() {

	/* read configuration */
	file, err := os.Open("config.json")
	defer file.Close()

	if err != nil {
		/* configure not found */
		log.Printf("Unable to read config.json. Setting default parameters.")
		conf.HttpPort = 8035
	} else {
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&conf)
		if err != nil {
			log.Printf("Error reading config.json: %s", err)
			return
		}
	}

	initialize()

	router := mux.NewRouter()
	// Each of these handler funcs would be called inside a go routine
	router.HandleFunc("/account", commonWrapper(OpenAccount)).Methods("POST")
	router.HandleFunc("/deposit", commonWrapper(Deposit)).Methods("POST")
	router.HandleFunc("/account/{aid:[0-9a-fA-F\\-]+}", commonWrapper(GetAccountInfo)).Methods("GET")
	router.HandleFunc("/transfer", commonWrapper(Transfer)).Methods("POST")
	router.HandleFunc("/su/{suid:[0-9a-fA-F\\-]+}/accounts", commonWrapper(GetAllAccounts)).Methods("GET")
	router.HandleFunc("/su/savestate", commonWrapper(SaveServerState)).Methods("POST")
	//router.HandleFunc("/su/get_users_of_feed/{suid:[0-9a-fA-F\\-]+}/{fid:[0-9a-fA-F\\-]+}", commonWrapper(GetUsersOfFeed)).Methods("GET")
	
	http.Handle("/", router)

	log.Println(fmt.Sprintf("Listening at port %d ...", conf.HttpPort))
	http.ListenAndServe(fmt.Sprintf(":%d", conf.HttpPort), router)
	log.Println("Done! Exiting...")

	uninitialize()
}
