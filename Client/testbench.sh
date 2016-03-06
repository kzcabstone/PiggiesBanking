
#!/bin/bash
num_of_clients=$1

export PATH=$PATH:$(pwd)

set -m # Enable Job Control

#######  create "IRS" account
client.sh -o 0

#######  open $num_of_clients accounts first
echo "Open $num_of_clients accounts"
for i in `seq $num_of_clients`; do # start $num_of_clients jobs in parallel
  client.sh -o $i &
done
# Wait for all parallel jobs to finish
while [ 1 ]; do fg 2> /dev/null; [ $? -eq 1 ] && break; done


######  deposit 200 to each account
echo "Deposit 200 into each account"
for i in `seq $num_of_clients`; do 
  client.sh -d $i 200 &
done
# Wait for all parallel jobs to finish
while [ 1 ]; do fg 2> /dev/null; [ $? -eq 1 ] && break; done

######  withdraw 100 from each account and transfer 1 to IRS
echo "Withdraw 100 from each account, and transfer 1 to account 0"
for i in `seq $num_of_clients`; do 
  client.sh -w $i 100 &
  client.sh -t $i 0 1 &
done
# Wait for all parallel jobs to finish
while [ 1 ]; do fg 2> /dev/null; [ $? -eq 1 ] && break; done

########### Now check the IRS account, it should have $100
echo "Now check the IRS account, it should have $num_of_clients"
client.sh -q 0
sleep 5

########### Now check all accounts, they should have $99
echo "Now check all accounts, they should have $ 99"
client.sh -i 15337
