package template

import "errors"

type MergedCoinPairs []Block

func (p *MergedCoinPairs) GetPrimary() Block {
	pair := *p
	return pair[0]
}

func (p *MergedCoinPairs) GetAux() (Block, error) {
	if len(*p) < 2 {
		return Block{}, errors.New("No aux coin in coin config")
	}
	pair := *p
	return pair[1], nil
}

func (p *MergedCoinPairs) GetAux2() (Block, error) {
	if len(*p) < 3 {
		return Block{}, errors.New("No aux 2 coin in coin config")
	}
	pair := *p
	return pair[2], nil
}
