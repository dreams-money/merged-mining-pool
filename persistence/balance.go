package persistence

import (
	"database/sql"
	"time"
)

type Balance struct {
	PoolID  string
	Coin    string
	Address string
	Amount  float32
	Created time.Time
	Updated time.Time
}

type BalanceChange struct {
	ID      uint
	PoolID  string
	Address string
	Amount  float32
	Usage   string
	Created time.Time
}

type BalanceRepository struct {
	*sql.DB
}

func (r *BalanceRepository) AddAmount(poolID, coin, address, usage string, amount float32) error {
	now := time.Now()

	query := "INSERT INTO balance_changes(poolid, coin, address, amount, usage, tags, created) "
	query = query + "VALUES(?, ?, ?, ?, ?, ?, ?)"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(&poolID, &coin, &address, &amount, &usage, "", &now)
	if err != nil {
		return err
	}

	balance, err := r.GetBalance(poolID, coin, address)
	if err != nil {
		return err
	}

	balanceRecord := Balance{
		PoolID:  poolID,
		Coin:    coin,
		Address: address,
		Created: now,
		Updated: now,
	}

	if balance == nil {
		return r.Insert(balanceRecord)
	}

	return r.Update(balanceRecord)
}

func (r *BalanceRepository) Insert(balance Balance) error {
	query := "INSERT INTO balances(poolid, address, amount, created, updated) "
	query = query + "VALUES(?, ?, ?, ?, ?)"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(balance.PoolID, balance.Address, balance.Amount,
		balance.Created, balance.Updated)
	return err
}

func (r *BalanceRepository) Update(balance Balance) error {
	query := "UPDATE balances SET amount = amount + ?, updated = now() at time zone 'utc' "
	query = query + "WHERE poolid = ? AND coin = ? AND address = ?"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(balance.Amount, balance.PoolID, balance.Coin, balance.Address)
	return err
}

func (r *BalanceRepository) GetBalance(poolID, coin, address string) (*float32, error) {
	query := "SELECT amount FROM balances WHERE poolid = ? AND coin = ? AND address = ?"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	row := stmt.QueryRow(poolID, coin, address)
	if row == nil {
		return nil, nil
	}

	var balance float32
	err = row.Scan(&balance)
	if err != nil {
		return nil, err
	}

	return &balance, nil
}

func (r *BalanceRepository) GetPoolBalancesOverThreshold(poolID string, minimum float32) ([]Balance, error) {
	query := "SELECT b.poolid, b.address, b.created, b.updated "
	query = query + "FROM balances b "
	query = query + "LEFT JOIN miner_settings ms ON ms.poolid = b.poolid AND ms.address = b.address "
	query = query + "WHERE b.poolid = ? AND b.amount >= COALESCE(ms.paymentthreshold, ?)"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(poolID, minimum)
	if err != nil {
		return nil, err
	}

	var balances []Balance
	for rows.Next() {
		var balance Balance

		err = rows.Scan(&balance)
		if err != nil {
			return nil, err
		}

		balances = append(balances, balance)
	}

	return balances, nil
}

func (r *PaymentRepository) PageBalanceChanges(poolID string, page, pageSize int) ([]BalanceChange, error) {
	query := "SELECT * FROM balance_changes WHERE poolid = @poolid "
	query = query + "ORDER BY created DESC OFFSET ? FETCH NEXT ? ROWS ONLY"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(poolID, page, pageSize)
	if err != nil {
		return nil, err
	}

	var changes []BalanceChange
	for rows.Next() {
		var change BalanceChange

		err = rows.Scan(&change)
		if err != nil {
			return nil, err
		}

		changes = append(changes, change)
	}

	return changes, nil
}

func (r *PaymentRepository) GetBalanceChangesCount(poolID string) (uint, error) {
	query := "SELECT COUNT(*) FROM balance_changes WHERE poolid = ?"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return 0, err
	}

	row := stmt.QueryRow(poolID)
	if row == nil {
		return 0, nil
	}

	var count uint
	err = row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
