Persistence Service
===================

POSTGRES, in my opinion, was the way to go due to it's high write throughput.

Currently, only POSTGRES is supported, but any storage engine (or combination of such) can be used to service the persistence layer.

Simply interface on the methods for each object, then factory out the connections with persister.go,

## Quick illustation of stats window adjustments
Accurate hashrates are possible with adjusted hashrate windows.

    adjustedWindow =

    A = startDifference
    B = endDifference

                       A          B
                       |          |
                       v          v
    windowStart ->  -                - <- now..
    minStat  -------|->  -      -  <-|--- maxStat
                    |    |______|    |
    <-not reachable |   STAT WINDOW  |  Not possible to reach -->
                    |________________|
                     HASHRATE WINDOW