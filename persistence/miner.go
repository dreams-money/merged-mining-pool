package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

const minerStatWindow = 20

type MinerRepository struct {
	*sql.DB
}

type MinerSettings struct {
	PoolID           string
	Miner            string
	PaymentThreshold float32
	Created          time.Time
	Updated          time.Time
}

func (r *MinerRepository) GetSettings(poolID, miner string) (MinerSettings, error) {
	var settings MinerSettings
	query := "SELECT poolid, address, paymentthreshold, created, updated FROM miner_settings WHERE poolid = $1 AND address = $2"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return settings, err
	}

	err = stmt.QueryRow(poolID, miner).Scan(&settings.PoolID, &settings.Miner,
		&settings.PaymentThreshold, &settings.Created, &settings.Updated)
	if err != nil {
		return settings, err
	}

	return settings, nil
}

func (r *MinerRepository) UpdateSettings(settings MinerSettings) error {
	query := "INSERT INTO miner_settings(poolid, address, paymentthreshold, created, updated) "
	query = query + "VALUES($1, $2, $3, now(), now()) "
	query = query + "ON CONFLICT ON CONSTRAINT miner_settings_pkey DO UPDATE "
	query = query + "SET paymentthreshold = $4, updated = now() "
	query = query + "WHERE miner_settings.poolid = $5 AND miner_settings.address = $6"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(settings.PoolID, settings.Miner, settings.PaymentThreshold,
		settings.PaymentThreshold, settings.PoolID, settings.Miner)
	return err
}

type MinerStat struct {
	PoolID          string
	Miner           string
	Worker          string
	Hashrate        float64
	SharesPerSecond float64
	Created         time.Time
}

func (r *MinerRepository) InsertMinerWorkerPerformanceStats(stat MinerStat) error {
	query := "INSERT INTO minerstats(poolid, miner, worker, hashrate, sharespersecond, created) "
	query = query + "VALUES($1, $2, $3, $4, $5, $6)"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(stat.PoolID, stat.Miner, stat.Worker, stat.Hashrate, stat.SharesPerSecond, stat.Created)
	return err
}

type WorkerStat struct {
	PoolID          string
	Miner           string
	Worker          string
	Hashrate        float64
	SharesPerSecond float64
	Status          string
	LastSeen        time.Time
	Rating          int64
	Created         time.Time
	Partition       int
}

type WorkersReport struct {
	PendingShares float64
	Created       time.Time
	Workers       map[string]WorkerStat
}

func newWorkerReport() WorkersReport {
	report := WorkersReport{}
	report.Workers = make(map[string]WorkerStat)
	return report
}

type MinerAccount struct {
	PendingBalance float32
	TotalPaid      float32
	TodayPaid      float32
	LastPayment    Payment
}

type ChainAccounts map[string]MinerAccount

func (accounts *ChainAccounts) GetPendingAmounts() map[string]float32 {
	amounts := make(map[string]float32)
	for chain, account := range *accounts {
		amounts[chain] = account.PendingBalance
	}
	return amounts
}

func (accounts *ChainAccounts) GetTotalPaidAmounts() map[string]float32 {
	amounts := make(map[string]float32)
	for chain, account := range *accounts {
		amounts[chain] = account.TodayPaid
	}
	return amounts
}

type MinerReport struct {
	Created time.Time
	ChainAccounts
	WorkersReport
}

func (r *MinerRepository) GetMinerStatsReport(poolID, address string, payments *PaymentRepository) (*MinerReport, error) {
	report := MinerReport{}
	report.WorkersReport.Workers = make(map[string]WorkerStat)

	shares := `SELECT SUM(difficulty) FROM shares WHERE poolid = $1 AND miner = $2`
	row := r.DB.QueryRow(shares, poolID, address)
	err := row.Scan(&report.PendingShares)
	if err != nil {
		return &report, err
	}

	var chain string
	var amount float32
	accounts := make(ChainAccounts)
	pendingBalances := `SELECT chain, amount FROM balances WHERE poolid = $1 AND address = $2`
	rows, err := r.DB.Query(pendingBalances, poolID, address)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		err = rows.Scan(&chain, &amount)
		if err != nil {
			return &report, err
		}
		account, found := accounts[chain]
		if !found {
			account = MinerAccount{}
		}

		account.PendingBalance = amount
		accounts[chain] = account
	}

	totalPaid := `select chain, sum(amount) AS amount from payments where poolid = $1 and address = $2 group by chain`
	rows, err = r.DB.Query(totalPaid, poolID, address)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		err = rows.Scan(&chain, &amount)
		if err != nil {
			return &report, err
		}
		account, found := accounts[chain]
		if !found {
			account = MinerAccount{}
		}

		account.TotalPaid = amount
		accounts[chain] = account
	}

	totalPaidToday := `select chain, sum(amount) AS amount from payments
	where poolid = $1 and address = $2 and created >= date_trunc('day', now())  group by chain`
	rows, err = r.DB.Query(totalPaidToday, poolID, address)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		err = rows.Scan(&chain, &amount)
		if err != nil {
			return &report, err
		}
		account, found := accounts[chain]
		if !found {
			account = MinerAccount{}
		}

		account.TodayPaid = amount
		accounts[chain] = account
	}

	lastPayments, err := payments.MinerLastPayments(poolID, address)
	if err != nil {
		return nil, err
	}
	for chain, payment := range lastPayments {
		account, found := accounts[chain]
		if !found {
			account = MinerAccount{}
		}

		account.LastPayment = payment
		accounts[chain] = account
	}

	report.ChainAccounts = accounts

	lastUpdate, err := r.LastStatUpdate(poolID, address)
	if err != nil {
		return nil, err
	}
	if lastUpdate != nil {
		// Ignore stale stats
		statCutoff := time.Now().Add(-1 * minerStatWindow * time.Minute)
		if lastUpdate.Before(statCutoff) {
			lastUpdate = nil
		}
	}

	if lastUpdate == nil {
		return &report, nil
	}

	stats, err := r.GetMinerStatsByCreatedTime(poolID, address, *lastUpdate)
	if err != nil {
		return nil, err
	}

	report.Created = *lastUpdate
	report.WorkersReport = newWorkerReport()

	for _, stat := range stats {
		report.WorkersReport.Workers[stat.Worker] = WorkerStat{
			Worker:          stat.Worker,
			Hashrate:        stat.Hashrate,
			SharesPerSecond: stat.SharesPerSecond,
		}
	}

	return &report, nil
}

type MinerAverage struct {
	Worker                 string    `json:"worker"`
	Created                time.Time `json:"created"`
	AverageHashrate        float64   `json:"avg_hashrate"`
	AverageSharesPerSecond float64   `json:"avg_sharespersecond"`
}

func (r *MinerRepository) GetMinerHourlyAveragesBetween(poolID, miner string, start, end time.Time) (map[string][]MinerAverage, error) {
	query := `SELECT worker,
		date_trunc('hour', created) AS created,
		AVG(hashrate) AS avg_hashrate,
		AVG(sharespersecond) AS sharespersecond
	FROM minerstats
	WHERE poolid = $1
	AND miner = $2
	AND created >= $3
	AND created <= $4
	GROUP BY date_trunc('hour', created), worker
	ORDER BY created, worker;`

	rows, err := r.DB.Query(query, poolID, miner, start, end)
	if err != nil {
		return nil, err
	}

	averages := make(map[string][]MinerAverage)
	for rows.Next() {
		var average MinerAverage
		err = rows.Scan(&average.Worker, &average.Created, &average.AverageHashrate, &average.AverageSharesPerSecond)
		if err != nil {
			return averages, err
		}
		averages[average.Worker] = append(averages[average.Worker], average)
	}

	return averages, nil
}

func (r *MinerRepository) GetMinerPerformanceBetweenTimeAtXMinuteIntervals(poolID, address string, start, end time.Time, xMinutes int) (*WorkersReport, error) {
	query := `SELECT date_trunc('hour', created) AS created,
	(extract(minute FROM created)::int / %v) AS partition,
	worker, AVG(hashrate) AS hashrate, AVG(sharespersecond) AS sharespersecond
	FROM minerstats
	WHERE poolid = $1 AND miner = $2 AND created >= $3 AND created <= $4
	GROUP BY 1, 2, worker
	ORDER BY 1, 2, worker`
	query = fmt.Sprintf(query, xMinutes)

	rows, err := r.DB.Query(query, poolID, address, start, end)
	if err != nil {
		return nil, err
	}

	report := newWorkerReport()
	for rows.Next() {
		var stat WorkerStat
		err = rows.Scan(&stat.Created, &stat.Partition, &stat.Worker, &stat.Hashrate, &stat.Hashrate)
		if err != nil {
			return &report, err
		}
		report.Workers[stat.Worker] = stat
	}

	return &report, nil
}

func (r *MinerRepository) GetMinerPerformanceBetweenTimesAtInterval(poolID, address string, start, end time.Time, interval time.Duration) (*WorkersReport, error) {
	intervalMap := make(map[time.Duration]string)
	intervalMap[time.Minute] = "minute"
	intervalMap[time.Hour] = "hour"
	intervalMap[time.Hour*24] = "day"

	intervalString, intervalExists := intervalMap[interval]
	if !intervalExists {
		return nil, errors.New("invalid input interval")
	}

	query := `SELECT poolid, miner, worker, date_trunc('%v', created) AS created, AVG(hashrate) AS hashrate,
	AVG(sharespersecond) AS sharespersecond FROM minerstats
	WHERE poolid = $1 AND miner = $2 AND created >= $3 AND created <= $4
	GROUP BY date_trunc('%v', created), worker
	ORDER BY created, worker;`
	query = fmt.Sprintf(query, intervalString, intervalString)

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(poolID, address, start, end)
	if err != nil {
		return nil, err
	}

	// We definitely massage the data a bit more..

	report := newWorkerReport()
	for rows.Next() {
		var stat MinerStat
		err = rows.Scan(&stat.PoolID, &stat.Miner, &stat.Worker, &stat.Created, &stat.Hashrate, &stat.SharesPerSecond)
		if err != nil {
			return &report, err
		}

		// report.Workers[stat.Worker] = stat
	}

	return &report, nil
}

func (r *MinerRepository) GetMinerStatsByCreatedTime(poolID, address string, created time.Time) ([]MinerStat, error) {
	query := "SELECT poolid, miner, worker, hashrate, sharespersecond, created FROM minerstats WHERE poolid = $1 AND miner = $2 AND created = $3"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	row, err := stmt.Query(poolID, address, created)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}

	var stats []MinerStat
	for row.Next() {
		var stat MinerStat
		err = row.Scan(&stat.PoolID, &stat.Miner, &stat.Worker, &stat.Hashrate, &stat.SharesPerSecond, &stat.Created)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func (r *MinerRepository) GetMinerStatsBetweenTimes(poolID, address string, start, end time.Time) ([]MinerStat, error) {
	return nil, nil
}

type MinerHashrates map[string]MinerStat // miner => MinerStat

func (r *MinerRepository) PageMinerHashrates(poolID string, from time.Time, page, pageSize int) (MinerHashrates, error) {
	minerHashrates := make(MinerHashrates)

	query := "WITH tmp AS "
	query = query + "("
	query = query + "	SELECT"
	query = query + "		ms.miner,"
	query = query + "		ms.hashrate,"
	query = query + "		ms.sharespersecond,"
	query = query + "		ROW_NUMBER() OVER(PARTITION BY ms.miner ORDER BY ms.hashrate DESC) AS rk"
	query = query + "	FROM (SELECT miner, SUM(hashrate) AS hashrate, SUM(sharespersecond) AS sharespersecond"
	query = query + "	   FROM minerstats"
	query = query + "	   WHERE poolid = $1 AND created >= $2 GROUP BY miner, created) ms"
	query = query + ") "
	query = query + "SELECT t.miner, t.hashrate, t.sharespersecond "
	query = query + "FROM tmp t "
	query = query + "WHERE t.rk = 1 "
	query = query + "ORDER by t.hashrate DESC "
	query = query + "OFFSET $3 FETCH NEXT $4 ROWS ONLY"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return minerHashrates, err
	}

	rows, err := stmt.Query(poolID, from, page, pageSize)
	if err != nil {
		return minerHashrates, err
	}
	for rows.Next() {
		var miner string
		var hashrate, sharespersecond float64

		err = rows.Scan(&miner, &hashrate, &sharespersecond)
		if err != nil {
			return minerHashrates, err
		}

		minerHashrates[miner] = MinerStat{
			Miner:           miner,
			Hashrate:        hashrate,
			SharesPerSecond: sharespersecond,
		}
	}

	return minerHashrates, nil
}

func (r *MinerRepository) DeleteMinerStatsBefore(date time.Time) error {
	query := "DELETE FROM minerstats WHERE created < $1"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(date)
	return err
}

func (r *MinerRepository) GetRecentyUsedIpAddresses(poolID, minerAddress string) ([]string, error) {
	query := "SELECT DISTINCT s.ipaddress FROM (SELECT ipaddress FROM shares WHERE poolid = $1 and miner = $2 ORDER BY CREATED DESC LIMIT 100) s"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(poolID, minerAddress)
	if err != nil {
		return nil, err
	}

	var ips []string
	for rows.Next() {
		var ip string
		err = rows.Scan(&ip)
		if err != nil {
			return ips, err
		}

		ips = append(ips, ip)
	}

	return ips, nil
}

func (r *MinerRepository) LastStatUpdate(poolID, miner string) (*time.Time, error) {
	query := "SELECT created FROM minerstats WHERE poolid = $1 AND miner = $2 "
	query = query + "ORDER BY created DESC LIMIT 1"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	row := stmt.QueryRow(poolID, miner)
	if row == nil {
		return nil, nil
	}

	var createdTime time.Time
	err = row.Scan(&createdTime)
	if err != nil {
		return nil, err
	}

	return &createdTime, nil
}

func (r *MinerRepository) GetWorkersLastSeen(poolID, miner string) (WorkersReport, error) {
	report := newWorkerReport()

	rows, err := r.DB.Query(`
	select worker, max(created)

	from public.shares

	where miner = $1

	group by worker

	having max(created) >= now() - INTERVAL '1 WEEKS'`, miner)
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	var workerID string
	var lastSeen time.Time
	now := time.Now()

	for rows.Next() {
		err = rows.Scan(&workerID, &lastSeen)
		if err != nil {
			return report, nil
		}

		lastSeenTimeDiff := now.Sub(lastSeen)
		status := "Active"
		if lastSeenTimeDiff.Minutes() >= 10 {
			status = "Inactive"
		}

		report.Workers[workerID] = WorkerStat{
			Status:   status,
			LastSeen: lastSeen,
		}
	}

	return report, nil
}
