
#!/bin/bash
export PATH=$PATH:$(pwd)
set -m # Enable Job Control

#######  create "IRS" account
client.sh -o 0

#######  open 100 accounts first
echo "Open 100 accounts"
for i in `seq 100`; do # start 100 jobs in parallel
  client.sh -o $i &
done
# Wait for all parallel jobs to finish
while [ 1 ]; do fg 2> /dev/null; [ $? == 1 ] && break; done


######  deposit 200 to each account
echo "Deposit 200 into each account"
for i in `seq 100`; do # start 30 jobs in parallel
  client.sh -d $i 200 &
done
# Wait for all parallel jobs to finish
while [ 1 ]; do fg 2> /dev/null; [ $? == 1 ] && break; done

######  withdraw 100 from each account and transfer 1 to IRS
echo "Withdraw 100 from each account, and transfer 1 to account 0"
for i in `seq 100`; do # start 30 jobs in parallel
  client.sh -w $i 100 &
  client.sh -t $i 0 1 &
done
# Wait for all parallel jobs to finish
while [ 1 ]; do fg 2> /dev/null; [ $? == 1 ] && break; done

########### Now check the IRS account, it should have $100
echo "Now check the IRS account, it should have $ 100"
client.sh -q 0
sleep 5

########### Now check all accounts, they should have $99
echo "Now check all accounts, they should have $ 99"
client.sh -i 15337
