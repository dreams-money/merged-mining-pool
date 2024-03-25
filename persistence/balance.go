package persistence

import (
	"database/sql"
	"time"
)

type Balance struct {
	PoolID  string
	Chain   string
	Address string
	Amount  float64
	Created time.Time
	Updated time.Time
}

type BalanceChange struct {
	ID      uint
	PoolID  string
	Chain   string
	Address string
	Amount  float32
	Usage   string
	Created time.Time
}

type BalanceRepository struct {
	*sql.DB
}

func (r *BalanceRepository) AddAmount(poolID, chain, address, usage string, amount float64) error {
	now := time.Now()

	// query := "INSERT INTO balance_changes(poolid, chain, address, amount, usage, tags, created) "
	query := `INSERT INTO balance_changes(poolid, chain, address, amount, usage, created)
				VALUES($1, $2, $3, $4, $5, $6)`

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	// _, err = stmt.Exec(poolID, chain, address, amount, usage, "", now)
	_, err = stmt.Exec(poolID, chain, address, amount, usage, now)
	if err != nil {
		return err
	}

	balance, err := r.GetBalance(poolID, chain, address)
	if err != nil {
		return err
	}

	balanceRecord := Balance{
		PoolID:  poolID,
		Chain:   chain,
		Address: address,
		Amount:  amount,
		Updated: now,
	}

	if balance == nil {
		balanceRecord.Created = now
		return r.Insert(balanceRecord)
	}

	return r.Update(balanceRecord)
}

func (r *BalanceRepository) Insert(balance Balance) error {
	query := "INSERT INTO balances(poolid, chain, address, amount, created, updated) "
	query = query + "VALUES($1, $2, $3, $4, $5, $6)"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(balance.PoolID, balance.Chain, balance.Address, balance.Amount,
		balance.Created, balance.Updated)
	return err
}

func (r *BalanceRepository) Update(balance Balance) error {
	query := "UPDATE balances SET amount = amount + $1, updated = now() at time zone 'utc' "
	query = query + "WHERE poolid = $2 AND chain = $3 AND address = $4"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(balance.Amount, balance.PoolID, balance.Chain, balance.Address)
	return err
}

func (r *BalanceRepository) GetBalance(poolID, chain, address string) (*float64, error) {
	query := "SELECT amount FROM balances WHERE poolid = $1 AND chain = $2 AND address = $3"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	row := stmt.QueryRow(poolID, chain, address)
	if row == nil {
		return nil, nil
	}

	var balance float64
	err = row.Scan(&balance)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &balance, nil
}

func (r *BalanceRepository) GetPoolBalancesOverThreshold(poolID, chain string, minimum float32) ([]Balance, error) {
	query := `SELECT b.poolid, b.chain, b.address, b.amount, b.created, b.updated
				FROM balances b
				LEFT JOIN miner_settings ms
				ON ms.poolid = b.poolid
				AND ms.address = b.address
				WHERE b.poolid = $1
				AND b.chain = $2
				AND b.amount >= COALESCE(ms.paymentthreshold, $3)`

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(poolID, chain, minimum)
	if err != nil {
		return nil, err
	}

	var balances []Balance
	for rows.Next() {
		var balance Balance

		err = rows.Scan(&balance.PoolID, &balance.Chain, &balance.Address, &balance.Amount, &balance.Created, &balance.Updated)
		if err != nil {
			return nil, err
		}

		balances = append(balances, balance)
	}

	return balances, nil
}

func (r *PaymentRepository) PageBalanceChanges(poolID string, page, pageSize int) ([]BalanceChange, error) {
	query := "SELECT * FROM balance_changes WHERE poolid = $1 "
	query = query + "ORDER BY created DESC OFFSET $2 FETCH NEXT $3 ROWS ONLY"

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
	query := "SELECT COUNT(*) FROM balance_changes WHERE poolid = $1"

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
