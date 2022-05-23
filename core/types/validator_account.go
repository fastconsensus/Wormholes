package types

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sort"
)

type Validator struct {
	Addr    common.Address
	Balance *big.Int
}

func (v *Validator) Address() common.Address {
	return v.Addr
}

func NewValidator(addr common.Address, balance *big.Int) *Validator {
	return &Validator{Addr: addr, Balance: balance}
}

type ValidatorList struct {
	Validators []*Validator
}

func NewValidatorList(validators []Validator) *ValidatorList {
	var validatorList *ValidatorList
	for i := 0; i < len(validators); i++ {
		validatorList.Validators = append(validatorList.Validators, &validators[i])
	}
	return validatorList
}

func (vl *ValidatorList) Len() int {
	return len(vl.Validators)
}

func (vl *ValidatorList) Less(i, j int) bool {
	return vl.Validators[i].Address().Hash().Big().Cmp(vl.Validators[j].Address().Hash().Big()) < 0
}

func (vl *ValidatorList) Swap(i, j int) {
	vl.Validators[i], vl.Validators[j] = vl.Validators[j], vl.Validators[i]
}

// 按距离升序排列
func (sl *ValidatorList) AddValidator(addr common.Address, balance *big.Int) bool {
	for _, v := range sl.Validators {
		if v.Address() == addr {
			v.Balance.Add(v.Balance, balance)
			sort.Sort(sl)
			return true
		}
	}
	sl.Validators = append(sl.Validators, NewValidator(addr, balance))
	sort.Sort(sl)
	return true
}

func (sl *ValidatorList) RemoveValidator(addr common.Address, balance *big.Int) bool {
	for i, v := range sl.Validators {
		if v.Address() == addr {
			if v.Balance.Cmp(balance) > 0 {
				v.Balance.Sub(v.Balance, balance)
				sort.Sort(sl)
				return true
			} else if v.Balance.Cmp(balance) == 0 {
				v.Balance.Sub(v.Balance, balance)
				sl.Validators = append(sl.Validators[:i], sl.Validators[i+1:]...)
				return true
			}
			sl.Validators = append(sl.Validators[:i], sl.Validators[i+1:]...)
			return true
		}
	}
	return false
}

// Query K validators closest to random numbers based on distance and pledge amount
func (sl *ValidatorList) ValidatorByDistanceAndWeight(addr []*big.Int, k int, randomHash common.Hash) []common.Address {
	// 地址转bigInt的最大值
	maxValue := common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff").Hash().Big()

	// 记录地址对应的权重
	addrToWeightMap := make(map[*big.Int]*big.Int, 0)

	// 哈希转160位地址
	r1 := randomHash[12:]
	x := common.BytesToAddress(r1).Hash().Big()

	for _, v := range addr {
		sub1 := big.NewInt(0)
		sub2 := big.NewInt(0)

		// 得到的sub1 和 sub2 是两个距离值，需要从中取小的
		sub1 = sub1.Sub(v, x)
		sub1 = sub1.Abs(sub1)
		sub2 = sub2.Sub(maxValue, sub1)
		if sub1.Cmp(sub2) < 0 {
			a := new(big.Int).Mul(sub1, sl.StakeBalance(common.BigToAddress(v)))
			w := new(big.Int).Div(a, sl.TotalStakeBalance())
			addrToWeightMap[v] = w
		} else {
			a := new(big.Int).Mul(sub2, sl.StakeBalance(common.BigToAddress(v)))
			w := new(big.Int).Div(a, sl.TotalStakeBalance())
			addrToWeightMap[v] = w
		}
	}

	sortMap := rankByWordCount(addrToWeightMap)
	res := make([]common.Address, 0)

	for i := 0; i < sortMap.Len(); i++ {
		if i < k {
			res = append(res, common.BigToAddress(sortMap[i].Key))
		} else {
			break
		}
	}
	return res
}

// Calculate the total amount of the stake account
func (sl *ValidatorList) TotalStakeBalance() *big.Int {
	var total = big.NewInt(0)
	for _, voter := range sl.Validators {
		total.Add(total, voter.Balance)
	}
	return total
}

// Returns the amount of the staked node
func (sl *ValidatorList) StakeBalance(address common.Address) *big.Int {
	for _, st := range sl.Validators {
		if st.Address().Hex() != address.Hex() {
			continue
		}
		return st.Balance
	}
	return big.NewInt(0)
}

func (sl *ValidatorList) ConvertToAddress() (addrs []common.Address) {
	for _, validator := range sl.Validators {
		addrs = append(addrs, validator.Addr)
	}
	return
}

func (sl *ValidatorList) ConvertToBigInt() (bigIntSlice []*big.Int) {
	for _, validator := range sl.Validators {
		bigIntSlice = append(bigIntSlice, validator.Addr.Hash().Big())
	}
	return
}

func (sl *ValidatorList) Exist(addr common.Address) bool {
	for _, v := range sl.Validators {
		if v.Addr != addr {
			continue
		}
		return true
	}
	return false
}
