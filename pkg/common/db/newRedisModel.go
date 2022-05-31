package db

import (
	"Open_IM/pkg/common/config"
	log2 "Open_IM/pkg/common/log"
	pbChat "Open_IM/pkg/proto/chat"
	pbCommon "Open_IM/pkg/proto/sdk_ws"
	"Open_IM/pkg/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"strconv"
	"time"
)

//func  (d *  DataBases)pubMessage(channel, msg string) {
//   d.rdb.Publish(context.Background(),channel,msg)
//}
//func  (d *  DataBases)pubMessage(channel, msg string) {
//	d.rdb.Publish(context.Background(),channel,msg)
//}

func (d *DataBases) NewGetMessageListBySeq(userID string, seqList []uint32, operationID string) (seqMsg []*pbCommon.MsgData, failedSeqList []uint32, errResult error) {
	for _, v := range seqList {
		//MESSAGE_CACHE:169.254.225.224_reliability1653387820_0_1
		key := messageCache + userID + "_" + strconv.Itoa(int(v))

		result, err := d.rdb.HGetAll(context.Background(), key).Result()
		if err != nil {
			errResult = err
			failedSeqList = append(failedSeqList, v)
			log2.NewWarn(operationID, "redis get message error:", err.Error(), v)
		} else {
			msg, err := Map2Pb(result)
			//msg := pbCommon.MsgData{}
			//err = jsonpb.UnmarshalString(result, &msg)
			if err != nil {
				errResult = err
				failedSeqList = append(failedSeqList, v)
				log2.NewWarn(operationID, "Unmarshal err", result, err.Error())
			} else {
				log2.NewDebug(operationID, "redis get msg is ", msg.String())
				seqMsg = append(seqMsg, msg)
			}

		}
	}
	return seqMsg, failedSeqList, errResult
}
func Map2Pb(m map[string]string) (*pbCommon.MsgData, error) {
	var data pbCommon.MsgData
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	err = proto.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (d *DataBases) NewSetMessageToCache(msgList []*pbChat.MsgDataToMQ, uid string, operationID string) error {
	ctx := context.Background()
	var failedList []pbChat.MsgDataToMQ
	for _, msg := range msgList {
		key := messageCache + uid + "_" + strconv.Itoa(int(msg.MsgData.Seq))
		s, err := utils.Pb2Map(msg.MsgData)
		if err != nil {
			log2.NewWarn(operationID, utils.GetSelfFuncName(), "Pb2Map failed", msg.MsgData.String(), uid, err.Error())
			continue
		}
		log2.NewDebug(operationID, "convert map is ", s)
		fmt.Println("ts", s)
		val, err := d.rdb.HMSet(ctx, key, s).Result()
		if err != nil {
			log2.NewWarn(operationID, utils.GetSelfFuncName(), "redis failed", "args:", key, *msg, uid, s, val)
			failedList = append(failedList, *msg)
		}
		d.rdb.Expire(ctx, key, time.Second*time.Duration(config.Config.MsgCacheTimeout))
	}
	if len(failedList) != 0 {
		return errors.New(fmt.Sprintf("set msg to cache failed, failed lists: %s", failedList))
	}
	return nil
}