/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package primitive

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nj-leegern/rocketmq-client-go/internal/utils"
)

const (
	PropertyKeySeparator                   = " "
	PropertyKeys                           = "KEYS"
	PropertyTags                           = "TAGS"
	PropertyWaitStoreMsgOk                 = "WAIT"
	PropertyDelayTimeLevel                 = "DELAY"
	PropertyRetryTopic                     = "RETRY_TOPIC"
	PropertyRealTopic                      = "REAL_TOPIC"
	PropertyRealQueueId                    = "REAL_QID"
	PropertyTransactionPrepared            = "TRAN_MSG"
	PropertyProducerGroup                  = "PGROUP"
	PropertyMinOffset                      = "MIN_OFFSET"
	PropertyMaxOffset                      = "MAX_OFFSET"
	PropertyBuyerId                        = "BUYER_ID"
	PropertyOriginMessageId                = "ORIGIN_MESSAGE_ID"
	PropertyTransferFlag                   = "TRANSFER_FLAG"
	PropertyCorrectionFlag                 = "CORRECTION_FLAG"
	PropertyMQ2Flag                        = "MQ2_FLAG"
	PropertyReconsumeTime                  = "RECONSUME_TIME"
	PropertyMsgRegion                      = "MSG_REGION"
	PropertyTraceSwitch                    = "TRACE_ON"
	PropertyUniqueClientMessageIdKeyIndex  = "UNIQ_KEY"
	PropertyMaxReconsumeTimes              = "MAX_RECONSUME_TIMES"
	PropertyConsumeStartTime               = "CONSUME_START_TIME"
	PropertyTranscationPreparedQueueOffset = "TRAN_PREPARED_QUEUE_OFFSET"
	PropertyTranscationCheckTimes          = "TRANSACTION_CHECK_TIMES"
	PropertyCheckImmunityTimeInSeconds     = "CHECK_IMMUNITY_TIME_IN_SECONDS"
	PropertyShardingKey                    = "SHARDING_KEY"
)

type Message struct {
	Topic         string
	Body          []byte
	Flag          int32
	TransactionId string
	Batch         bool
	// QueueID is the queue that messages will be sent to. the value must be set if want to custom the queue of message,
	// just ignore if not.
	Queue *MessageQueue

	properties map[string]string
	mutex      sync.RWMutex
}

func (m *Message) WithProperty(key, value string) {
	if key == "" || value == "" {
		return
	}
	m.mutex.Lock()
	if m.properties == nil {
		m.properties = make(map[string]string)
	}
	m.properties[key] = value
	m.mutex.Unlock()
}

func (m *Message) GetProperty(key string) string {
	m.mutex.RLock()
	v := m.properties[key]
	m.mutex.RUnlock()
	return v
}

func (m *Message) RemoveProperty(key string) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	value, exist := m.properties[key]
	if !exist {
		return ""
	}
	delete(m.properties, key)
	return value
}
func (m *Message) MarshallProperties() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	buffer := bytes.NewBufferString("")
	for k, v := range m.properties {
		buffer.WriteString(k)
		buffer.WriteRune(nameValueSeparator)
		buffer.WriteString(v)
		buffer.WriteRune(propertySeparator)
	}
	return buffer.String()
}

// unmarshalProperties parse data into property kv pairs.
func (m *Message) UnmarshalProperties(data []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.properties == nil {
		m.properties = make(map[string]string)
	}
	items := bytes.Split(data, []byte{propertySeparator})
	for _, item := range items {
		kv := bytes.Split(item, []byte{nameValueSeparator})
		if len(kv) == 2 {
			m.properties[string(kv[0])] = string(kv[1])
		}
	}
}

func NewMessage(topic string, body []byte) *Message {
	msg := &Message{
		Topic:      topic,
		Body:       body,
		properties: make(map[string]string),
	}
	//msg.properties[PropertyWaitStoreMsgOk] = strconv.FormatBool(true)
	return msg
}

// WithDelayTimeLevel set message delay time to consume.
// reference delay level definition: 1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m 10m 20m 30m 1h 2h
// delay level starts from 1. for example, if we set param level=1, then the delay time is 1s.
func (m *Message) WithDelayTimeLevel(level int) *Message {
	m.WithProperty(PropertyDelayTimeLevel, strconv.Itoa(level))
	return m
}

func (m *Message) WithTag(tags string) *Message {
	m.WithProperty(PropertyTags, tags)
	return m
}

func (m *Message) WithKeys(keys []string) *Message {
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(PropertyKeySeparator)
	}

	m.WithProperty(PropertyKeys, sb.String())
	return m
}

func (m *Message) WithShardingKey(key string) *Message {
	m.WithProperty(PropertyShardingKey, key)
	return m
}

func (m *Message) GetTags() string {
	return m.GetProperty(PropertyTags)
}

func (m *Message) GetKeys() string {
	return m.GetProperty(PropertyKeys)
}

func (m *Message) GetShardingKey() string {
	return m.GetProperty(PropertyShardingKey)
}

func (m *Message) String() string {
	return fmt.Sprintf("[topic=%s, body=%s, Flag=%d, properties=%v, TransactionId=%s]",
		m.Topic, string(m.Body), m.Flag, m.properties, m.TransactionId)
}

type MessageExt struct {
	Message
	MsgId                     string
	QueueId                   int32
	StoreSize                 int32
	QueueOffset               int64
	SysFlag                   int32
	BornTimestamp             int64
	BornHost                  string
	StoreTimestamp            int64
	StoreHost                 string
	CommitLogOffset           int64
	BodyCRC                   int32
	ReconsumeTimes            int32
	PreparedTransactionOffset int64
}

func (msgExt *MessageExt) GetTags() string {
	return msgExt.GetProperty(PropertyTags)
}

func (msgExt *MessageExt) GetRegionID() string {
	return msgExt.GetProperty(PropertyMsgRegion)
}

func (msgExt *MessageExt) IsTraceOn() string {
	return msgExt.GetProperty(PropertyTraceSwitch)
}

func (msgExt *MessageExt) String() string {
	return fmt.Sprintf("[Message=%s, MsgId=%s, QueueId=%d, StoreSize=%d, QueueOffset=%d, SysFlag=%d, "+
		"BornTimestamp=%d, BornHost=%s, StoreTimestamp=%d, StoreHost=%s, CommitLogOffset=%d, BodyCRC=%d, "+
		"ReconsumeTimes=%d, PreparedTransactionOffset=%d]", msgExt.Message.String(), msgExt.MsgId, msgExt.QueueId,
		msgExt.StoreSize, msgExt.QueueOffset, msgExt.SysFlag, msgExt.BornTimestamp, msgExt.BornHost,
		msgExt.StoreTimestamp, msgExt.StoreHost, msgExt.CommitLogOffset, msgExt.BodyCRC, msgExt.ReconsumeTimes,
		msgExt.PreparedTransactionOffset)
}

func DecodeMessage(data []byte) []*MessageExt {
	msgs := make([]*MessageExt, 0)
	buf := bytes.NewBuffer(data)
	count := 0
	for count < len(data) {
		msg := &MessageExt{}

		// 1. total size
		binary.Read(buf, binary.BigEndian, &msg.StoreSize)
		count += 4

		// 2. magic code
		buf.Next(4)
		count += 4

		// 3. body CRC32
		binary.Read(buf, binary.BigEndian, &msg.BodyCRC)
		count += 4

		// 4. queueID
		binary.Read(buf, binary.BigEndian, &msg.QueueId)
		count += 4

		// 5. Flag
		binary.Read(buf, binary.BigEndian, &msg.Flag)
		count += 4

		// 6. QueueOffset
		binary.Read(buf, binary.BigEndian, &msg.QueueOffset)
		count += 8

		// 7. physical offset
		binary.Read(buf, binary.BigEndian, &msg.CommitLogOffset)
		count += 8

		// 8. SysFlag
		binary.Read(buf, binary.BigEndian, &msg.SysFlag)
		count += 4

		// 9. BornTimestamp
		binary.Read(buf, binary.BigEndian, &msg.BornTimestamp)
		count += 8

		// 10. born host
		hostBytes := buf.Next(4)
		var port int32
		binary.Read(buf, binary.BigEndian, &port)
		msg.BornHost = fmt.Sprintf("%s:%d", utils.GetAddressByBytes(hostBytes), port)
		count += 8

		// 11. store timestamp
		binary.Read(buf, binary.BigEndian, &msg.StoreTimestamp)
		count += 8

		// 12. store host
		hostBytes = buf.Next(4)
		binary.Read(buf, binary.BigEndian, &port)
		msg.StoreHost = fmt.Sprintf("%s:%d", utils.GetAddressByBytes(hostBytes), port)
		count += 8

		// 13. reconsume times
		binary.Read(buf, binary.BigEndian, &msg.ReconsumeTimes)
		count += 4

		// 14. prepared transaction offset
		binary.Read(buf, binary.BigEndian, &msg.PreparedTransactionOffset)
		count += 8

		// 15. body
		var length int32
		binary.Read(buf, binary.BigEndian, &length)
		msg.Body = buf.Next(int(length))
		if (msg.SysFlag & FlagCompressed) == FlagCompressed {
			msg.Body = utils.UnCompress(msg.Body)
		}
		count += 4 + int(length)

		// 16. topic
		_byte, _ := buf.ReadByte()
		msg.Topic = string(buf.Next(int(_byte)))
		count += 1 + int(_byte)

		// 17. properties
		var propertiesLength int16
		binary.Read(buf, binary.BigEndian, &propertiesLength)
		if propertiesLength > 0 {
			msg.UnmarshalProperties(buf.Next(int(propertiesLength)))
		}
		count += 2 + int(propertiesLength)

		msg.MsgId = createMessageId(hostBytes, port, msg.CommitLogOffset)
		//count += 16
		if msg.properties == nil {
			msg.properties = make(map[string]string, 0)
		}
		msgs = append(msgs, msg)
	}

	return msgs
}

// MessageQueue message queue
type MessageQueue struct {
	Topic      string `json:"topic"`
	BrokerName string `json:"brokerName"`
	QueueId    int    `json:"queueId"`
}

func (mq *MessageQueue) String() string {
	return fmt.Sprintf("MessageQueue [topic=%s, brokerName=%s, queueId=%d]", mq.Topic, mq.BrokerName, mq.QueueId)
}

func (mq *MessageQueue) HashCode() int {
	result := 1
	result = 31*result + utils.HashString(mq.BrokerName)
	result = 31*result + mq.QueueId
	result = 31*result + utils.HashString(mq.Topic)

	return result
}

func (mq MessageQueue) Equals(queue *MessageQueue) bool {
	// TODO
	return mq.BrokerName == queue.BrokerName && mq.Topic == queue.Topic && mq.QueueId == mq.QueueId
}

type AccessChannel int

const (
	// connect to private IDC cluster.
	Local AccessChannel = iota
	// connect to Cloud service.
	Cloud
)

type MessageType int

const (
	NormalMsg MessageType = iota
	TransMsgHalf
	TransMsgCommit
	DelayMsg
)

type LocalTransactionState int

const (
	CommitMessageState LocalTransactionState = iota + 1
	RollbackMessageState
	UnknowState
)

type TransactionListener interface {
	//  When send transactional prepare(half) message succeed, this method will be invoked to execute local transaction.
	ExecuteLocalTransaction(Message) LocalTransactionState

	// When no response to prepare(half) message. broker will send check message to check the transaction status, and this
	// method will be invoked to get local transaction status.
	CheckLocalTransaction(MessageExt) LocalTransactionState
}

type MessageID struct {
	Addr   string
	Port   int
	Offset int64
}

func createMessageId(addr []byte, port int32, offset int64) string {
	buffer := new(bytes.Buffer)
	buffer.Write(addr)
	binary.Write(buffer, binary.BigEndian, port)
	binary.Write(buffer, binary.BigEndian, offset)
	return strings.ToUpper(hex.EncodeToString(buffer.Bytes()))
}

func UnmarshalMsgID(id []byte) (*MessageID, error) {
	if len(id) < 32 {
		return nil, fmt.Errorf("%s len < 32", string(id))
	}
	var (
		ipBytes     = make([]byte, 4)
		portBytes   = make([]byte, 4)
		offsetBytes = make([]byte, 8)
	)
	hex.Decode(ipBytes, id[0:8])
	hex.Decode(portBytes, id[8:16])
	hex.Decode(offsetBytes, id[16:32])

	return &MessageID{
		Addr:   utils.GetAddressByBytes(ipBytes),
		Port:   int(binary.BigEndian.Uint32(portBytes)),
		Offset: int64(binary.BigEndian.Uint64(offsetBytes)),
	}, nil
}

var (
	CompressedFlag = 0x1

	MultiTagsFlag = 0x1 << 1

	TransactionNotType = 0

	TransactionPreparedType = 0x1 << 2

	TransactionCommitType = 0x2 << 2

	TransactionRollbackType = 0x3 << 2
)

func GetTransactionValue(flag int) int {
	return flag & TransactionRollbackType
}

func ResetTransactionValue(flag int, typeFlag int) int {
	return (flag & (^TransactionRollbackType)) | typeFlag
}

func ClearCompressedFlag(flag int) int {
	return flag & (^CompressedFlag)
}

var (
	counter        int16 = 0
	startTimestamp int64 = 0
	nextTimestamp  int64 = 0
	prefix         string
	locker         sync.Mutex
	classLoadId    int32 = 0
)

func init() {
	buf := new(bytes.Buffer)

	ip, err := utils.ClientIP4()
	if err != nil {
		ip = utils.FakeIP()
	}
	_, _ = buf.Write(ip)
	_ = binary.Write(buf, binary.BigEndian, Pid())
	_ = binary.Write(buf, binary.BigEndian, classLoadId)
	prefix = strings.ToUpper(hex.EncodeToString(buf.Bytes()))
}

func CreateUniqID() string {
	locker.Lock()
	defer locker.Unlock()

	if time.Now().Unix() > nextTimestamp {
		updateTimestamp()
	}
	counter++
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, int32((time.Now().Unix()-startTimestamp)*1000))
	_ = binary.Write(buf, binary.BigEndian, counter)

	return prefix + hex.EncodeToString(buf.Bytes())
}

func updateTimestamp() {
	year, month := time.Now().Year(), time.Now().Month()
	startTimestamp = time.Date(year, month, 1, 0, 0, 0, 0, time.Local).Unix()
	nextTimestamp = time.Date(year, month, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0).Unix()
}

func Pid() int16 {
	return int16(os.Getpid())
}
