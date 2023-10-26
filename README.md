Merged Mining Pool
==================

![Dogecoin Logo](https://user-images.githubusercontent.com/5210627/256921635-3b7c1d9e-0148-4953-890e-5f57758973a4.png)
A merged mining pool centered around Doge/Litecoin
![Litecoin Logo](https://user-images.githubusercontent.com/5210627/256921657-11899bf5-995b-47ce-b7af-f7ee03d4da32.png)

Features
--------
  - Stratum
  - Subscribes to block notifications from blockchain daemon
  - Unique extranonce generation
  - Merged mining

Todo
----
  - Single coin mining
  - RPC failover
  - API service
  - Payouts service
  - Variable difficulty

Getting Started
---------------

Take a look at config.example.json to get started.  You'll need access to a [blockchain RPC](https://dogecoin.com/dogepedia/how-tos/operating-a-node/) and a [ZMQ block notification URL](https://github.com/bitcoin/bitcoin/blob/master/doc/zmq.md).

You have to start your nodes with block notification on:

    // Assuming your node is on the same machine as this pool
    -zmqpubhashblock="tcp://127.0.0.1:<your-port-here>"

    // remote nodes need additional configuration if they're on WAN or LAN (firewalls, port forwarding, etc.)
    -zmqpubhashblock="tcp://0.0.0.0:<your-port-here>"

Connecting to the pool
----------------------

Once you have it running, your client can connect with the following login:

  - username: youPrimaryCoinMinerAddress-yourAux1CoinMinerAddress.rigID
  - password: none