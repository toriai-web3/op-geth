package api

import (
	"fmt"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"sync"
	"time"
)

type Transaction struct {
	client      *rpc.Client
	lock        sync.Mutex
	dailSuccess bool
	pool        *TxPool
}

func NewTransaction() *Transaction {
	return &Transaction{
		pool: NewTxPool(),
	}
}

func (t *Transaction) SendRawTransaction(input hexutil.Bytes) (r string, err error) {
	defer func() {
		fmt.Println("SendRawTransaction", r, err)
	}()

	fmt.Println("SendRawTransaction input", input)
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return "fail", err
	}
	fmt.Println(tx)

	t.pool.Push(input)
	return "success", nil
}

func (t *Transaction) dail() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if !t.dailSuccess {
		client, err := rpc.Dial("http://localhost:8545")
		if err != nil {
			panic("rpc.Dial err:" + err.Error())
		}
		t.client = client
		t.dailSuccess = true
	}
}

func (t *Transaction) NewBlock(headBlock common.Hash) (r string, err error) {
	defer func() {
		fmt.Println("NewBlock", r, err)
	}()

	t.dail()

	gasLimit := uint64(300000000)
	pops := t.pool.Pop(math.MaxInt32)

	txData := make([][]byte, 0)
	for _, v := range pops {
		txData = append(txData, v)
	}

	fc := &engine.ForkchoiceStateV1{
		HeadBlockHash: headBlock,
	}
	attributes := &engine.PayloadAttributes{
		Timestamp:             uint64(time.Now().UnixNano()),
		Random:                common.HexToHash("0x962d15f88c4bb703c8dde604cf820cb4962d15f88c4bb703c8dde604cf820cb4"),
		SuggestedFeeRecipient: common.HexToAddress("0x4200000000000000000000000000000000000011"),
		Transactions:          txData,
		NoTxPool:              true,
		GasLimit:              &gasLimit,
	}

	var payloadId engine.ForkChoiceResponse
	err = t.client.Call(&payloadId, "engine_forkchoiceUpdatedV1", fc, attributes)
	if err != nil {
		return "fail", err
	}
	fmt.Println("payloadId", payloadId)

	var payload engine.ExecutableData
	err = t.client.Call(&payload, "engine_getPayloadV1", payloadId.PayloadID)
	if err != nil {
		return "fail", err
	}
	fmt.Println("payload", payload)

	var payloadStatus engine.PayloadStatusV1
	err = t.client.Call(&payloadStatus, "engine_newPayloadV1", payload)
	if err != nil {
		return "fail", err
	}
	fmt.Println("payloadStatus", payloadStatus)

	var payloadId2 engine.ForkChoiceResponse
	fc2 := &engine.ForkchoiceStateV1{
		HeadBlockHash:      *payloadStatus.LatestValidHash,
		SafeBlockHash:      *payloadStatus.LatestValidHash,
		FinalizedBlockHash: *payloadStatus.LatestValidHash,
	}
	err = t.client.Call(&payloadId2, "engine_forkchoiceUpdatedV1", fc2, nil)
	if err != nil {
		return "fail", err
	}
	fmt.Println("payloadId2", payloadId2)

	return "success", nil
}
