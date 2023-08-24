package template

import (
	"errors"
)

type TemplatesHistory struct {
	templates []MergedCoinPairs
	MaxLength int
}

func NewTemplatesHistory(length int) TemplatesHistory {
	return TemplatesHistory{
		MaxLength: length,
	}
}

func (history *TemplatesHistory) AddMergedCoinTemplatePairs(newPairs MergedCoinPairs) {
	if len(history.templates) <= history.MaxLength {
		history.templates = append([]MergedCoinPairs{newPairs}, history.templates...)
		return
	}
	history.templates = append([]MergedCoinPairs{newPairs}, history.templates[:history.MaxLength-1]...)
}

func (history *TemplatesHistory) GetLatest() (MergedCoinPairs, error) {
	return history.GetSpecific(0)
}

func (history *TemplatesHistory) GetSpecific(index int) (MergedCoinPairs, error) {
	if len(history.templates) < 1 || index > len(history.templates)-1 {
		return nil, errors.New("History index out of range")
	}

	return history.templates[index], nil
}
