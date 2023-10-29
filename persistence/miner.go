package persistence

import (
	"database/sql"
	"log"
	"time"
)

const minerStatWindow = 20

type MinerStat struct {
	PoolID          string
	Miner           string
	Worker          string
	Hashrate        float64
	SharesPerSecond float64
	Created         time.Time
}

type WorkerStat struct {
	PoolID          string
	Miner           string
	Worker          string
	Hashrate        float64
	SharesPerSecond float64
	Status          string
	LastSeen        string
	Rating          int64
	Created         time.Time
}

type WorkersReport struct {
	Created time.Time
	Workers map[string]WorkerStat
}

func newWorkerReport() WorkersReport {
	report := WorkersReport{}
	report.Workers = make(map[string]WorkerStat)
	return report
}

type MinerReport struct {
	Created        time.Time
	PendingShares  float64
	PendingBalance float32
	TotalPaid      float32
	TodayPaid      float32
	LastPayment    *Payment
	WorkersReport
}

type MinerSettings struct {
	PoolID           string
	Miner            string
	PaymentThreshold float32
	Created          time.Time
	Updated          time.Time
}

type MinerRepository struct {
	*sql.DB
}

func (r *MinerRepository) GetSettings(poolID, miner string) (MinerSettings, error) {
	var settings MinerSettings
	query := "SELECT poolid, address, paymentthreshold, created, updated FROM miner_settings WHERE poolid = ? AND address = ?"

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
	query = query + "VALUES(?, ?, ?, now(), now()) "
	query = query + "ON CONFLICT ON CONSTRAINT miner_settings_pkey DO UPDATE "
	query = query + "SET paymentthreshold = ?, updated = now() "
	query = query + "WHERE miner_settings.poolid = ? AND miner_settings.address = ?"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(settings.PoolID, settings.Miner, settings.PaymentThreshold,
		settings.PaymentThreshold, settings.PoolID, settings.Miner)
	return err
}

func (r *MinerRepository) InsertMinerWorkerPerformanceStats(stat MinerStat) error {
	query := "INSERT INTO minerstats(poolid, miner, worker, hashrate, sharespersecond, created) "
	query = query + "VALUES(?, ?, ?, ?, ?, ?)"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(&stat.PoolID, &stat.Miner, &stat.Worker, &stat.Hashrate,
		&stat.SharesPerSecond, &stat.Created)
	return err
}

func (r *MinerRepository) GetMinerStatsReport(poolID, address string, payments *PaymentRepository) (*MinerReport, error) {
	query := "SELECT (SELECT SUM(difficulty) FROM shares WHERE poolid = ? AND miner = ?) AS pendingshares, "
	query = query + "(SELECT amount FROM balances WHERE poolid = ? AND address = ?) AS pendingbalance, "
	query = query + "(SELECT SUM(amount) FROM payments WHERE poolid = ? and address = ?) AS totalpaid, "
	query = query + "(SELECT SUM(amount) FROM payments WHERE poolid = ? and address = ? and created >= date_trunc('day', now())) AS todaypaid"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	report := MinerReport{}

	row := stmt.QueryRow(poolID, address, poolID, address, poolID, address, poolID, address)
	if row == nil {
		return nil, nil
	}

	err = row.Scan(&report.PendingShares, &report.PendingBalance, &report.TotalPaid, &report.TodayPaid)
	if err != nil {
		return nil, err
	}

	report.LastPayment, err = payments.MinerLastPayment(poolID, address)
	if err != nil {
		return nil, err
	}

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

	stat, err := r.GetMinerStatByCreatedTime(poolID, address, *lastUpdate)
	if err != nil {
		return nil, err
	}

	report.Created = *lastUpdate
	report.WorkersReport.Workers[stat.Worker] = WorkerStat{
		Worker:          stat.Worker,
		Hashrate:        stat.Hashrate,
		SharesPerSecond: stat.SharesPerSecond,
	}

	return &report, nil
}

func (r *MinerRepository) GetMinerStatByCreatedTime(poolID, address string, created time.Time) (*MinerStat, error) {
	var stat MinerStat
	query := "SELECT poolid, miner, worker, hashrate, sharespersecond, created FROM minerstats WHERE poolid = ? AND miner = ? AND created = ?"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	row := stmt.QueryRow(poolID, address, created)
	if row == nil {
		return nil, nil
	}

	err = row.Scan(&stat.PoolID, &stat.Miner, &stat.Worker, &stat.Hashrate, &stat.SharesPerSecond, &stat.Created)
	if err != nil {
		return nil, err
	}

	return &stat, nil
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
	query = query + "	   WHERE poolid = ? AND created >= ? GROUP BY miner, created) ms"
	query = query + ") "
	query = query + "SELECT t.miner, t.hashrate, t.sharespersecond "
	query = query + "FROM tmp t "
	query = query + "WHERE t.rk = 1 "
	query = query + "ORDER by t.hashrate DESC "
	query = query + "OFFSET ? FETCH NEXT ? ROWS ONLY"

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
	query := "DELETE FROM minerstats WHERE created < @date"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(date)
	return err
}

func (r *MinerRepository) GetRecentyUsedIpAddresses(poolID, minerAddress string) ([]string, error) {
	query := "SELECT DISTINCT s.ipaddress FROM (SELECT ipaddress FROM shares WHERE poolid = ? and miner = ? ORDER BY CREATED DESC LIMIT 100) s"

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
	query := "SELECT created FROM minerstats WHERE poolid = ? AND miner = ? "
	query = query + "ORDER BY created DESC LIMIT 1"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	row := stmt.QueryRow(poolID, miner)
	if row == nil {
		return nil, nil
	}

	var createdString string
	err = row.Scan(createdString)
	if err != nil {
		return nil, err
	}

	createdTime, err := time.Parse("2006-01-02 15:04:05", createdString)
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
	var lastSeen string
	now := time.Now()

	for rows.Next() {
		err = rows.Scan(&workerID, &lastSeen)
		if err != nil {
			return report, nil
		}

		lastSeenTime, err := time.Parse(time.RFC3339, lastSeen)
		if err != nil {
			return report, nil
		}
		lastSeenTimeDiff := now.Sub(lastSeenTime)
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
