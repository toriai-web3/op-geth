package api

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type TxPool struct {
	txs []hexutil.Bytes
}

func NewTxPool() *TxPool {
	return &TxPool{
		txs: make([]hexutil.Bytes, 0),
	}
}

func (t *TxPool) Push(bytes hexutil.Bytes) {
	t.txs = append(t.txs, bytes)
}

func (t *TxPool) Pop(n int) []hexutil.Bytes {
	if n > len(t.txs) {
		n = len(t.txs)
	}
	r := t.txs[:n]
	t.txs = t.txs[n:]
	return r
}
