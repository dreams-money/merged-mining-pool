package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type PoolStat struct {
	ID                   uint
	PoolID               string
	ConnectedMiners      uint
	ConnectedWorkers     uint
	PoolHashrate         float64
	NetworkHashrate      float64
	NetworkDifficulty    float64
	LastNetworkBlockTime time.Time
	BlockHeight          uint
	ConnectedPeers       uint
	SharesPerSecond      float64
	Created              time.Time
}

type PoolRepository struct {
	*sql.DB
}

func (r *PoolRepository) InsertPoolStat(stat PoolStat) error {
	query := "INSERT INTO poolstats(poolid, connectedminers, connectedworkers, poolhashrate, networkhashrate, networkdifficulty, "
	query = query + "lastnetworkblocktime, blockheight, connectedpeers, sharespersecond, created) "
	query = query + "VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(stat.PoolID, stat.ConnectedMiners, stat.ConnectedWorkers, stat.PoolHashrate,
		stat.NetworkHashrate, stat.NetworkDifficulty, stat.LastNetworkBlockTime, stat.BlockHeight,
		stat.ConnectedPeers, stat.SharesPerSecond, stat.Created)
	return err
}

func (r *PoolRepository) GetLastStat(poolID string) (PoolStat, error) {
	stat := PoolStat{}
	query := "SELECT poolid, connectedminers, connectedworkers, poolhashrate, sharespersecond, networkhashrate, networkdifficulty, "
	query = query + "lastnetworkblocktime, blockheight, connectedpeers, created "
	query = query + "FROM poolstats WHERE poolid = $1 ORDER BY created DESC FETCH NEXT 1 ROWS ONLY"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return stat, err
	}

	err = stmt.QueryRow(poolID).Scan(&stat.PoolID, &stat.ConnectedMiners, &stat.ConnectedWorkers, &stat.PoolHashrate,
		&stat.SharesPerSecond, &stat.NetworkHashrate, &stat.NetworkDifficulty, &stat.LastNetworkBlockTime,
		&stat.BlockHeight, &stat.ConnectedPeers, &stat.Created)

	return stat, err
}

func (r *PoolRepository) TotalPoolPayments(poolID string) (float32, error) {
	query := "SELECT sum(amount) FROM payments WHERE poolid = $1"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return 0, err
	}

	var totalPayment float32
	err = stmt.QueryRow(poolID).Scan(&totalPayment)
	if err != nil {
		return 0, err
	}

	return totalPayment, nil
}

func (r *PoolRepository) PoolPerformanceBetween(poolID string, start, end time.Time, sampleInterval time.Duration) ([]PoolStat, error) {
	trunc := ""
	if sampleInterval == time.Hour {
		trunc = "hour"
	} else if sampleInterval == time.Hour*24 {
		trunc = "day"
	} else {
		return nil, errors.New("unknown sample interval")
	}

	query := "SELECT date_trunc('%v', created) AS created, "
	query = query + "AVG(poolhashrate) AS poolhashrate, AVG(networkhashrate) AS networkhashrate, "
	query = query + "AVG(networkdifficulty) AS networkdifficulty, "
	query = query + "CAST(AVG(connectedminers) AS BIGINT) AS connectedminers "

	query = query + "FROM poolstats "
	query = query + "WHERE poolid = $1 AND created >= $2 AND created <= $3 "
	query = query + "GROUP BY date_trunc('%v', created) "
	query = query + "ORDER BY created"

	query = fmt.Sprint(query, trunc)

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	var stats []PoolStat
	rows, err := stmt.Query(poolID, start, end)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var stat PoolStat

		err = rows.Scan(&stat.Created, &stat.PoolHashrate, &stat.NetworkHashrate, &stat.NetworkDifficulty,
			&stat.ConnectedMiners)
		if err != nil {
			return stats, err
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

type MinerWorkerHashrates map[string]map[string]float64 // miner => worker => hashrate

func (results *MinerWorkerHashrates) GroupByMiner() map[string][]float64 {
	miners := make(map[string][]float64)
	for miner, workers := range *results {
		hashrates, exists := miners[miner]
		if !exists {
			var rates []float64
			hashrates = rates
		}
		for _, worker := range workers {
			hashrates = append(hashrates, worker)
		}

		miners[miner] = hashrates
	}

	return miners
}

func (r *PoolRepository) MinerWorkerHashrates(poolID string) (MinerWorkerHashrates, error) {
	minerWorkerHashrates := make(MinerWorkerHashrates)

	query := "SELECT s.miner, s.worker, s.hashrate FROM "
	query = query + "( "
	query = query + "	WITH cte AS"
	query = query + "	("
	query = query + "		SELECT"
	query = query + "			ROW_NUMBER() OVER (partition BY miner, worker ORDER BY created DESC) as rk,"
	query = query + "			miner, worker, hashrate"
	query = query + "		FROM minerstats"
	query = query + "		WHERE poolid = $1"
	query = query + "	)"
	query = query + "	SELECT miner, worker, hashrate"
	query = query + "	FROM cte"
	query = query + "	WHERE rk = 1"
	query = query + ") s "
	query = query + "WHERE s.hashrate > 0;"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return minerWorkerHashrates, err
	}

	rows, err := stmt.Query(poolID)
	if err != nil {
		return minerWorkerHashrates, err
	}

	for rows.Next() {
		var miner, worker string
		var hashrate float64

		_, exists := minerWorkerHashrates[miner]
		if !exists {
			workerHashrates := make(map[string]float64)
			minerWorkerHashrates[miner] = workerHashrates
		}

		err = rows.Scan(&miner, &worker, &hashrate)
		if err != nil {
			return minerWorkerHashrates, err
		}

		minerWorkerHashrates[miner][worker] = hashrate
	}

	return minerWorkerHashrates, nil
}

func (r *PoolRepository) DeletePoolStatsBefore(date time.Time) error {
	query := "DELETE FROM poolstats WHERE created < $1"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(date)
	return err
}
