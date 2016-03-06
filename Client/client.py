import argparse, sys
import requests, json

def usage():
	print ("Example usage: ")
	print ("		user -o 25   ==>  open account 25")
	print ("		user -q 25   ==>  query info on account 25")
	print (" 	    user -d 25 3000   ==>  deposit $3000 to account 25")
	print ("		user -w 25 3000   ==>  withdraw $3000 from account 25")
	print ("		user -t 25 50 3000	  ==>  transfer $3000 from account 25 to account 50")
	print ("		user -i 15337   ==>  As a super user, list all accounts")
	print ("		user -i 15337 -s  ==>  As a super user, tell the server to save its state to file")
	print ("		user -a 192.168.0.1 -o 25 ==>  instead of default localhost:8035, connects to server 192.168.0.1 and open account 25")

def OpenAccount(ip, aid):
	print ("Open account {0}".format(aid))
	
	req_json = {"aid": aid}
	path = "{0}/account".format(ip.rstrip('\\'));
	resp = requests.post(path, data=json.dumps(req_json),
                     headers={'Content-Type':'application/json'})
	if resp.status_code not in (200, 201):
		print("Status code", resp.status_code)
	print(resp.json())

def Deposit(ip, aid, amount):
	print ("Deposit {0} to account {1}".format(amount, aid))
	path = "{0}/deposit".format(ip.rstrip('\\'));
	req_json = {"aid": aid, "amount": amount }
	resp = requests.post(path, data=json.dumps(req_json),
                     headers={'Content-Type':'application/json'})
	if resp.status_code not in (200, 201):
		print("Status code", resp.status_code)
	print(resp.json())

def Withdraw(ip, aid, amount):
	print ("Withdraw {0} from account {1}".format(amount, aid))
	path = "{0}/deposit".format(ip.rstrip('\\'));
	req_json = {"aid": aid, "amount": str(-1 * int(amount)) }
	resp = requests.post(path, data=json.dumps(req_json),
                     headers={'Content-Type':'application/json'})
	if resp.status_code not in (200, 201):
		print("Status code", resp.status_code)
	print(resp.json())

def Transfer(ip, aid, aid0, amount):
	print ("Transfer {0} from account {1} to account {2}".format(amount, aid, aid0))
	path = "{0}/transfer".format(ip.rstrip('\\'));
	req_json = {"aid": aid, "amount": amount, "aid0": aid0 }
	resp = requests.post(path, data=json.dumps(req_json),
                     headers={'Content-Type':'application/json'})
	if resp.status_code not in (200, 201):
		print("Status code", resp.status_code)
	print(resp.json())

def GetAccount(ip, aid):
	path = "{0}/account/{1}".format(ip.rstrip('\\'), aid)
	resp = requests.get(path)
	if resp.status_code != 200:
		print("Status code", resp.status_code)
	print('{}'.format(resp.json()["accountstr"]))

def GetAllAccounts(ip, suid):
	print("Get all accounts")
	path = "{0}/su/{1}/accounts".format(ip.rstrip('\\'), suid)
	resp = requests.get(path)
	if resp.status_code != 200:
		print("Status code", resp.status_code)
	print(resp.json())

def saveServerState(ip, suid):
	print ("Save server state: ")	
	path = "{0}/su/savestate".format(ip.rstrip('\\'));
	req_json = {"suid": suid}
	resp = requests.post(path, data=json.dumps(req_json), headers={'Content-Type':'application/json'})
	if resp.status_code not in (200, 201):
		print("Status code", resp.status_code)
	print(resp.json())

def main():
	argparser = argparse.ArgumentParser(description="Piggies' banking client")
	argparser.add_argument("-a", "--addr", help="server address", default="http://localhost:8035")
	argparser.add_argument("-s", "--savestate", help="tell server to save its state", action='store_true')
	argparser.add_argument('-i', '--iamsu', help='as superuser <id>, list all accounts')
	group = argparser.add_mutually_exclusive_group()
	group.add_argument("-o", "--open", help='open <account>')
	group.add_argument("-q", '--query', help='query <account>')
	group.add_argument('-d', '--deposit', nargs=2, help='deposit <account> <amount>')
	group.add_argument('-w', '--withdraw', nargs=2, help='withdraw <account> <amount>')
	group.add_argument('-t', '--transfer', nargs=3, help='transfer <account_from> <account_to> <amount>')

	args = argparser.parse_args()

	if args.open:
		OpenAccount(args.addr, args.open)
	elif args.deposit:
		Deposit(args.addr, args.deposit[0], args.deposit[1])
	elif args.withdraw:
		Withdraw(args.addr, args.withdraw[0], args.withdraw[1])
	elif args.transfer:
		Transfer(args.addr, args.transfer[0], args.transfer[1], args.transfer[2])
	elif args.query:
		GetAccount(args.addr, args.query)
	elif args.iamsu:
		suid = args.iamsu
		if args.savestate:
			saveServerState(args.addr, suid)
		else:
			GetAllAccounts(args.addr, suid)
	else:
		usage()
	return 0


if __name__ == "__main__":
	rc = main()
	sys.exit(rc)