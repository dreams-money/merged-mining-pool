package persistence

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type Payment struct {
	ID                          uint
	PoolID                      string
	Chain                       string
	Address                     string
	Amount                      float64
	TransactionConfirmationData string
	Created                     time.Time
}

type PaymentRepository struct {
	*sql.DB
}

func (r *PaymentRepository) Insert(payment Payment) error {
	query := "INSERT INTO payments(poolid, chain, address, amount, transactionconfirmationdata, created) "
	query = query + "VALUES($1, $2, $3, $4, $5, $6)"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(&payment.PoolID, &payment.Chain, &payment.Address, &payment.Amount,
		&payment.TransactionConfirmationData, &payment.Created)
	return err
}

func (r *PaymentRepository) InsertBatch(payments []Payment) error {
	txn, err := r.DB.Begin()
	if err != nil {
		return err
	}

	fields := pq.CopyIn("poolid", "chain", "address", "amount", "transactionconfirmationdata", "created")
	stmt, err := txn.Prepare(fields)
	if err != nil {
		return err
	}

	for _, payment := range payments {
		_, err = stmt.Exec(payment.PoolID, payment.Chain, payment.Address, payment.Amount,
			payment.TransactionConfirmationData, payment.Created)
		if err != nil {
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	return txn.Commit()
}

func (r *PaymentRepository) PagePayments(poolID, miner string, page, pageSize int) ([]Payment, error) {
	query := "SELECT poolid, chain, address, amount, transactionconfirmationdata, created FROM payments WHERE poolid = $1 "
	if miner != "" {
		query = query + " AND address = $4 "
	}
	query = query + "ORDER BY created DESC OFFSET $2 FETCH NEXT $3 ROWS ONLY"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	var payments []Payment
	var rows *sql.Rows
	if miner == "" {
		rows, err = stmt.Query(poolID, page, pageSize)
	} else {
		rows, err = stmt.Query(poolID, page, pageSize, miner)
	}
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var payment Payment

		err = rows.Scan(&payment.PoolID, &payment.Chain, &payment.Address, &payment.Amount,
			&payment.TransactionConfirmationData, &payment.Created)
		if err != nil {
			return payments, err
		}

		payments = append(payments, payment)
	}

	return payments, nil
}

func (r *PaymentRepository) PageMinerPaymentsByDay(poolID, miner string, page, pageSize int) ([]Payment, error) {
	query := "SELECT SUM(amount) AS amount, date_trunc('day', created) AS date FROM payments WHERE poolid = $1 "
	if miner != "" {
		query = query + " AND address = $4 "
	}
	query = query + "GROUP BY date ORDER BY date DESC OFFSET $2 FETCH NEXT $3 ROWS ONLY"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	var payments []Payment
	var rows *sql.Rows
	if miner == "" {
		rows, err = stmt.Query(poolID, page, pageSize)
	} else {
		rows, err = stmt.Query(poolID, page, pageSize, miner)
	}
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var payment Payment

		err = rows.Scan(&payment.PoolID, &payment.Chain, &payment.Address, &payment.Amount,
			&payment.TransactionConfirmationData, &payment.Created)
		if err != nil {
			return payments, err
		}

		payments = append(payments, payment)
	}

	return payments, nil
}

func (r *PaymentRepository) PaymentsCount(poolID, miner string) (uint, error) {
	query := "SELECT COUNT(*) FROM payments WHERE poolid = $1"
	if miner != "" {
		query = query + " AND address = $2 "
	}

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return 0, err
	}

	var count uint
	if miner == "" {
		err = stmt.QueryRow(poolID).Scan(&count)
	} else {
		err = stmt.QueryRow(poolID, miner).Scan(&count)
	}
	if err != nil {
		return 0, err
	}

	return count, nil
}

type PaymentsCountByDay map[string]uint

func (r *PaymentRepository) MinerPaymentsByDayCount(poolID, miner string) (PaymentsCountByDay, error) {
	paymentsByDay := make(PaymentsCountByDay)

	query := "SELECT COUNT(*) FROM (SELECT SUM(amount) AS amount, date_trunc('day', created) AS date "
	query = query + "FROM payments WHERE poolid = $1 "
	query = query + "AND address = $2 "
	query = query + "FROM GROUP BY date "
	query = query + "ORDER BY date DESC) s"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(poolID, miner)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var date string
		var count uint

		err = rows.Scan(&count, &date)
		if err != nil {
			return nil, err
		}

		paymentsByDay[date] = count
	}

	return paymentsByDay, nil
}

func (r *PaymentRepository) MinerLastPayments(poolID, miner string) (map[string]Payment, error) {
	query := `SELECT poolid, chain, address, amount, transactionconfirmationdata, created

			FROM payments

			WHERE poolid = $1 AND address = $2
			AND created = (
				SELECT max(b.created)
				from payments b
				where b.poolid = payments.poolid
				and b.chain = payments.chain
			)`

	rows, err := r.DB.Query(query, poolID, miner)
	if rows == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	payments := make(map[string]Payment)
	for rows.Next() {
		var payment Payment
		err = rows.Scan(&payment.PoolID, &payment.Chain, &payment.Address,
			&payment.Amount, &payment.TransactionConfirmationData, &payment.Created)
		if err != nil {
			return nil, err
		}
		payments[payment.Chain] = payment
	}

	return payments, nil
}
