package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	StatusPending   = "pending"
	StatusOrphaned  = "orphaned"
	StatusConfirmed = "confirmed"
)

type Found struct {
	ID                          uint
	PoolID                      string
	Chain                       string
	BlockHeight                 uint
	NetworkDifficulty           float64
	Status                      string
	Type                        string
	ConfirmationProgress        float32
	Effort                      float64
	TransactionConfirmationData string
	Miner                       string
	Reward                      float64
	Source                      string
	Hash                        string
	Created                     time.Time
}

type FoundBlocks []Found

func (b *FoundBlocks) GetConfirmed() FoundBlocks {
	var confirmed FoundBlocks
	for _, block := range *b {
		if block.Status == StatusConfirmed {
			confirmed = append(confirmed, block)
		}
	}
	return confirmed
}

type FoundRepository struct {
	*sql.DB
}

func (r *FoundRepository) Insert(block Found) error {
	query := `INSERT INTO blocks(poolid, chain, blockheight, networkdifficulty, status, "type", transactionconfirmationdata, miner, reward, effort, confirmationprogress, source, hash, created)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	block.NetworkDifficulty = roundToThreeDigits(block.NetworkDifficulty)

	_, err := r.DB.Exec(query, &block.PoolID, &block.Chain, &block.BlockHeight, &block.NetworkDifficulty,
		&block.Status, &block.Type, &block.TransactionConfirmationData, &block.Miner,
		&block.Reward, &block.Effort, &block.ConfirmationProgress, &block.Source, &block.Hash, &block.Created)

	return err
}

func (r *FoundRepository) Update(block Found) error {
	query := "UPDATE blocks SET blockheight = $1, status = $2, type = $3, "
	query = query + "reward = $4, effort = $5, "
	query = query + "confirmationprogress = $6, hash = $7 "
	query = query + "WHERE id = $8"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	result, err := stmt.Exec(block.BlockHeight, block.Status, block.Type, block.Reward, block.Effort,
		block.ConfirmationProgress, block.Hash, block.ID)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count < 1 {
		m := fmt.Sprintf("No update of block: %v", block.BlockHeight)
		return errors.New(m)
	}

	return err
}

func (r *FoundRepository) Delete(block Found) error {
	query := "DELETE FROM blocks WHERE id = $1"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(block.ID)
	return err
}

func (r *FoundRepository) PageBlocks(poolID, chain string, blockStatus []string, page, pageSize int) ([]Found, error) {
	query := `SELECT id, poolid, chain, blockheight, networkdifficulty, status, type, confirmationprogress,
	          effort, transactionconfirmationdata, miner, reward, source, hash, created
			  FROM blocks WHERE poolid = $1 AND chain = $2 AND status = ANY($3)
			  ORDER BY created DESC OFFSET $4 FETCH NEXT $5 ROWS ONLY`

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	statusString := strings.Join(blockStatus, ",")
	statusString = "{" + statusString + "}"

	rows, err := stmt.Query(poolID, chain, statusString, page, pageSize)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}

	var blockPage []Found
	for rows.Next() {
		var block Found

		err = rows.Scan(&block.ID, &block.PoolID, &block.Chain, &block.BlockHeight, &block.NetworkDifficulty,
			&block.Status, &block.Type, &block.ConfirmationProgress, &block.Effort, &block.TransactionConfirmationData,
			&block.Miner, &block.Reward, &block.Source, &block.Hash, &block.Created)
		if err != nil {
			return nil, err
		}

		blockPage = append(blockPage, block)
	}

	return blockPage, nil
}

func (r *FoundRepository) PageBlocksAcrossAllPools(blockStatus uint, page, pageSize int) ([]Found, error) {
	query := "SELECT poolid, blockheight, networkdifficulty, status, type, confirmationprogress, "
	query = query + "effort, transactionconfirmationdata, miner, reward, source, hash, created "
	query = query + "FROM blocks WHERE status = ANY($1) "
	query = query + "ORDER BY created DESC OFFSET $2 FETCH NEXT $3 ROWS ONLY"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(blockStatus, page, pageSize)
	if err != nil {
		return nil, err
	}

	var blockPage []Found
	for rows.Next() {
		var block Found

		err = rows.Scan(&block.PoolID, &block.BlockHeight, &block.NetworkDifficulty, &block.Status, &block.Type,
			&block.ConfirmationProgress, &block.Effort, &block.TransactionConfirmationData, &block.Miner,
			&block.Reward, &block.Source, &block.Hash, &block.Created)
		if err != nil {
			return nil, err
		}

		blockPage = append(blockPage, block)
	}

	return blockPage, nil
}

func (r *FoundRepository) PendingBlocksForPool(poolID string) (FoundBlocks, error) {
	query := `SELECT id, poolid, type, chain, blockheight, networkdifficulty, status,
					confirmationprogress, effort, transactionconfirmationdata,
					miner, reward, source, hash, created
		 		FROM blocks WHERE poolid = $1 AND status = $2`

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(poolID, StatusPending)
	if err != nil {
		return nil, err
	}

	var pending FoundBlocks
	for rows.Next() {
		var block Found

		err = rows.Scan(&block.ID, &block.PoolID, &block.Type, &block.Chain,
			&block.BlockHeight, &block.NetworkDifficulty, &block.Status,
			&block.ConfirmationProgress, &block.Effort,
			&block.TransactionConfirmationData, &block.Miner, &block.Reward,
			&block.Source, &block.Hash, &block.Created)
		if err != nil {
			return nil, err
		}

		pending = append(pending, block)
	}

	return pending, nil
}

func (r *FoundRepository) BlockBefore(poolID string, blockStatus []string, before time.Time) (*Found, error) {
	query := `SELECT poolid, blockheight, networkdifficulty, status, type, confirmationprogress,
				effort, transactionconfirmationdata, miner, reward, source, hash, created

				FROM blocks WHERE poolid = $1 AND status = ANY($2) AND created < $3
				ORDER BY created DESC FETCH NEXT 1 ROWS ONLY`

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	statusString := strings.Join(blockStatus, ",")
	statusString = "{" + statusString + "}"

	row := stmt.QueryRow(poolID, statusString, before)
	if row == nil {
		return nil, nil
	}

	found := Found{}
	err = row.Scan(&found.PoolID, &found.BlockHeight, &found.NetworkDifficulty, &found.Status, &found.Type,
		&found.ConfirmationProgress, &found.Effort, &found.TransactionConfirmationData, &found.Miner,
		&found.Reward, &found.Source, &found.Hash, &found.Created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return &found, err
	}

	return &found, nil
}

func (r *FoundRepository) BlockByHeight(poolID string, height uint) (*Found, error) {
	query := `SELECT poolid, blockheight, networkdifficulty, status, type, confirmationprogress,
			effort, transactionconfirmationdata, miner, reward, source, hash, created
			FROM blocks WHERE poolid = $1 AND blockheight = $2`

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	row := stmt.QueryRow(poolID)
	if row == nil {
		return nil, nil
	}

	var found Found
	err = row.Scan(&found)
	if err != nil {
		return &found, err
	}

	return &found, nil
}

func (r *FoundRepository) PoolBlockCount(poolID string) (uint, error) {
	query := "SELECT COUNT(*) FROM blocks WHERE poolid = $1"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return 0, err
	}

	var count uint
	err = stmt.QueryRow(poolID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *FoundRepository) PoolBlocksPerHour(poolID string) (uint, error) {
	query := `SELECT count(*)

	FROM public.blocks

	WHERE poolid = $1
	AND created >= (now() - INTERVAL '1 HOURS')`

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return 0, err
	}

	var count uint
	err = stmt.QueryRow(poolID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *FoundRepository) PoolLastBlockTime(poolID string) (*time.Time, error) {
	query := "SELECT created FROM blocks WHERE poolid = $1 ORDER BY created DESC LIMIT 1"

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	row := stmt.QueryRow(poolID)
	if row == nil {
		return nil, nil
	}

	var created time.Time
	err = row.Scan(&created)
	if err != nil {
		return &created, err
	}

	return &created, nil
}
