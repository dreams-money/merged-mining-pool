package persistence

import (
	"database/sql"
	"fmt"

	"designs.capital/dogepool/config"
)

var (
	Balances BalanceRepository
	Blocks   FoundRepository
	Miners   MinerRepository
	Payments PaymentRepository
	Pool     PoolRepository
	Shares   ShareRepository
)

func MakePersister(configuration *config.Config) error {
	connStr := "postgres://%v:%v@%v:%v/%v?sslmode=%v"
	config := configuration.Persister
	connStr = fmt.Sprintf(connStr, config.User, config.Password,
		config.Host, config.Port, config.Database, config.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	Balances = BalanceRepository{db}
	Blocks = FoundRepository{db}
	Miners = MinerRepository{db}
	Payments = PaymentRepository{db}
	Pool = PoolRepository{db}
	Shares = ShareRepository{db}

	return nil
}
