package internal

import (
	"fmt"
	"sort"

	"github.com/tuannh982/dag-bft/dag/commons"
	"github.com/tuannh982/dag-bft/utils/collections"
)

type ValidatorSet interface {
	NewWaveIfNotExists(w commons.Wave)
	GetWave(w commons.Wave) collections.Set[commons.Address]
	String() string
}

type validatorSet struct {
	internal map[commons.Wave]collections.Set[commons.Address]
}

func NewValidatorSet() ValidatorSet {
	return &validatorSet{
		internal: make(map[commons.Wave]collections.Set[commons.Address]),
	}
}

func addressHash(a commons.Address) string {
	return string(a)
}

func (s *validatorSet) NewWaveIfNotExists(w commons.Wave) {
	if _, ok := s.internal[w]; !ok {
		s.internal[w] = collections.NewHashSet(addressHash)
	}
}

func (s *validatorSet) GetWave(w commons.Wave) collections.Set[commons.Address] {
	return s.internal[w]
}

func (s *validatorSet) String() string {
	m := make(map[commons.Wave][]string)
	waveSet := make([]commons.Wave, 0, len(s.internal))
	for wave, addresses := range s.internal {
		ss := make([]string, 0)
		for _, address := range addresses.Entries() {
			ss = append(ss, fmt.Sprint(address))
		}
		m[wave] = ss
		waveSet = append(waveSet, wave)
	}
	sort.Slice(waveSet, func(i, j int) bool {
		return waveSet[i] < waveSet[j]
	})
	ret := ""
	for _, wave := range waveSet {
		ret += fmt.Sprintf("%d:%s\n", wave, m[wave])
	}
	return ret
}
