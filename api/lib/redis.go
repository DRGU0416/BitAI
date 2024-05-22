package lib

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

var (
	RDB *redis.Client

	RedisNull = "redis: nil"
)

var (
	RedisPrefix = "camera:"

	//短信验证码
	RedisMessagePhone = RedisPrefix + "smss:%s"    // 短信验证码 手机号 60s/次
	RedisMessageIP    = RedisPrefix + "smss:%s:%s" // 0611:ip 100次/d
	RedisMessageValue = RedisPrefix + "smsv:%s"    // 验证码

	//token
	RedisUserToken = RedisPrefix + "utoken" // Hash key:UserID value:md5(token)

	// CDN任务
	RedisCDNList = RedisPrefix + "task:cdn"

	// SD任务队列
	RedisSDList           = RedisPrefix + "task:sd"          // SD任务队列
	RedisSDCardList       = RedisPrefix + "task:card"        // SD分身图片任务队列
	RedisSDPhotoList      = RedisPrefix + "task:photo"       // 写真快任务队列
	RedisSDPhotoSlowList  = RedisPrefix + "task:photo:slow"  // 写真慢任务队列
	RedisSDOSSList        = RedisPrefix + "task:sdoss"       // OSS任务队列
	RedisSDPhotoHrList    = RedisPrefix + "task:photo:hr"    // 写真高清任务队列
	RedisSDCheckFrontList = RedisPrefix + "task:check:front" // 检测正面照任务队列
	RedisSDCheckSideList  = RedisPrefix + "task:check:side"  // 检测侧面照任务队列

	// 维护任务状态
	RedisTaskRecordHash = RedisPrefix + "task:record"

	// 每日注册人数
	RedisUserRegCount = RedisPrefix + "reg:count:%s" // reg:count:20230719
	// 每日每人写真任务数
	RedisUserPhotoCountHash = RedisPrefix + "photo:count:%s" // key=photo:count:20230719 field=用户ID value=写真任务数
	// 支付订单并发控制
	RedisOrderLock = RedisPrefix + "order:%s" // order_num

	// 账号错误统计
	RedisSDAccountError = RedisPrefix + "account:error:sd" // SD账号错误统计

	// 模板使用统计
	RedisTemplateSet = RedisPrefix + "template:%d"
)

const (
	REC_FRONT = 1 // 正面照
	REC_SIDE  = 2 // 侧面照
	REC_LORA  = 3 // 训练
	REC_CARD  = 4 // 分身
	REC_PHOTO = 5 // 写真
	REC_HR    = 6 // 高清
)

// 专用于维护任务队列
type TaskRecord struct {
	TaskType  int   // 1-正面照检测 2-侧面照检测 3-Lora训练 4-分身任务 5-写真任务 6-高清
	TaskID    int   //
	ExpiredAt int64 // 超时时间
	TryTimes  int   // 尝试次数
}

func (r *TaskRecord) Set() error {
	if err := r.Get(); err != nil && err != redis.Nil {
		return err
	}

	switch r.TaskType {
	case REC_FRONT:
		r.ExpiredAt = time.Now().Add(time.Minute).Unix()
	case REC_SIDE:
		r.ExpiredAt = time.Now().Add(time.Minute * 2).Unix()
	case REC_LORA:
		r.ExpiredAt = time.Now().Add(time.Minute * 30).Unix()
	case REC_CARD:
		r.ExpiredAt = time.Now().Add(time.Minute).Unix()
	case REC_PHOTO:
		r.ExpiredAt = time.Now().Add(time.Minute).Unix()
	case REC_HR:
		r.ExpiredAt = time.Now().Add(time.Minute).Unix()
	}
	r.TryTimes++

	record, err := json.MarshalToString(r)
	if err != nil {
		return err
	}
	return RDB.HSet(ctx, RedisTaskRecordHash, fmt.Sprintf("%d_%d", r.TaskType, r.TaskID), record).Err()
}

func (r *TaskRecord) ClearExpireAt() error {
	r.ExpiredAt = 0
	record, err := json.MarshalToString(r)
	if err != nil {
		return err
	}
	return RDB.HSet(ctx, RedisTaskRecordHash, fmt.Sprintf("%d_%d", r.TaskType, r.TaskID), record).Err()
}

func (r *TaskRecord) Get() error {
	record, err := RDB.HGet(ctx, RedisTaskRecordHash, fmt.Sprintf("%d_%d", r.TaskType, r.TaskID)).Result()
	if err != nil {
		return err
	}
	return json.UnmarshalFromString(record, r)
}

func (r *TaskRecord) Delete() error {
	return RDB.HDel(ctx, RedisTaskRecordHash, fmt.Sprintf("%d_%d", r.TaskType, r.TaskID)).Err()
}

func GetTaskRecords() (map[string]string, error) {
	return RDB.HGetAll(ctx, RedisTaskRecordHash).Result()
}

func init() {
	config := viper.GetStringMapString("redis")
	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config["host"], config["port"]),
		Password: config["password"],
		DB:       0,
	})
}

func SetMobileCodeEX(mobile, code string, expire time.Duration) (string, error) {
	return RDB.SetEX(ctx, fmt.Sprintf(RedisMessageValue, mobile), code, expire).Result()
}

func GetMobileCode(mobile string) (string, error) {
	return RDB.Get(ctx, fmt.Sprintf(RedisMessageValue, mobile)).Result()
}

// 加入SD队列
func PushSDTask(id int) error {
	return RDB.LPush(ctx, RedisSDList, id).Err()
}

// 获取SD队列
func PopSDTask() (int, error) {
	value, err := RDB.RPop(ctx, RedisSDList).Result()
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// 查看训练任务ID排名
func GetSDTaskRank(id int) int {
	strId := strconv.Itoa(id)
	vals := RDB.LRange(ctx, RedisSDList, 0, -1).Val()
	length := len(vals)
	if length == 0 {
		return 0
	}

	for i, v := range vals {
		if v == strId {
			return length - i
		}
	}
	return 0
}

// 加入SD队列
func PushSDCardTask(id int) error {
	return RDB.LPush(ctx, RedisSDCardList, id).Err()
}

// 获取SD队列
func PopSDCardTask() (int, error) {
	value, err := RDB.RPop(ctx, RedisSDCardList).Result()
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// 加入写真队列
func PushSDPhotoTask(id int, slow bool) error {
	if slow {
		return RDB.LPush(ctx, RedisSDPhotoSlowList, id).Err()
	}
	return RDB.LPush(ctx, RedisSDPhotoList, id).Err()
}

// 获取写真队列
func PopSDPhotoTask() (int, error) {
	value, err := RDB.RPop(ctx, RedisSDPhotoList).Result()
	if err != nil {
		if err != redis.Nil {
			return 0, err
		}
		value, err = RDB.RPop(ctx, RedisSDPhotoSlowList).Result()
		if err != nil {
			return 0, err
		}
	}
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// 从写真队列删除
func DelSDPhotoTask(id int) error {
	if err := RDB.LRem(ctx, RedisSDPhotoList, 1, id).Err(); err != nil {
		return err
	}
	return RDB.LRem(ctx, RedisSDPhotoSlowList, 1, id).Err()
}

// 加入正面照检测队列
func PushSDCheckFrontTask(ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	strids, err := json.MarshalToString(ids)
	if err != nil {
		return err
	}

	return RDB.LPush(ctx, RedisSDCheckFrontList, strids).Err()
}

// 加入侧面照检测队列
func PushSDCheckSideTask(ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	strids, err := json.MarshalToString(ids)
	if err != nil {
		return err
	}

	return RDB.LPush(ctx, RedisSDCheckSideList, strids).Err()
}

// 获取检测队列 1:正面 2:侧面
func PopSDCheckTask() (int, []int, error) {
	ctype := 1
	value, err := RDB.RPop(ctx, RedisSDCheckFrontList).Result()
	if err != nil {
		if err != redis.Nil {
			return 0, nil, err
		}
		value, err = RDB.RPop(ctx, RedisSDCheckSideList).Result()
		if err != nil {
			return 0, nil, err
		}
		ctype = 2
	}
	var ids []int
	if err = json.UnmarshalFromString(value, &ids); err != nil {
		return 0, nil, err
	}
	return ctype, ids, nil
}

// 加入高清写真队列
func PushSDPhotoHrTask(id int) error {
	return RDB.LPush(ctx, RedisSDPhotoHrList, id).Err()
}

// 获取高清写真队列
func PopSDPhotoHrTask() (int, error) {
	value, err := RDB.RPop(ctx, RedisSDPhotoHrList).Result()
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// 加入SD OSS队列
func PushSDOSSTask(id int) error {
	return RDB.LPush(ctx, RedisSDOSSList, id).Err()
}

// 获取SD OSS队列
func PopSDOSSTask() (int, error) {
	value, err := RDB.RPop(ctx, RedisSDOSSList).Result()
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// SD账号异常
func SDAccountError(id int) (int64, error) {
	return RDB.HIncrBy(ctx, RedisSDAccountError, strconv.Itoa(id), 1).Result()
}

// 重置MJ账号
func ResetSDAccount(id int) error {
	return RDB.HDel(ctx, RedisSDAccountError, strconv.Itoa(id)).Err()
}

// 检查元素是否存在List中
func CheckListExist(key string, id uint) bool {
	value := strconv.Itoa(int(id))
	vals := RDB.LRange(ctx, key, 0, -1).Val()
	for _, val := range vals {
		if val == value {
			return true
		}
	}
	return false
}

const (
	CDN_HEAD   int = 1 // 头像
	CDN_DELETE int = 2 // 删除CDN单张图片
	TRAIN_SIDE int = 3 // 删除训练补充图片(用户上传图片)
)

type UploadCDNTask struct {
	TaskType int      `json:"task_type"` // 1:头像
	TaskId   int      `json:"task_id"`
	DelPath  []string `json:"del_path"`
}

// 加入上传CDN处理队列
func PushCDNTask(t *UploadCDNTask) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	return RDB.LPush(ctx, RedisCDNList, string(data)).Err()
}

// 获取上传CDN处理队列
func PopCDNTask() (UploadCDNTask, error) {
	task := UploadCDNTask{}
	value, err := RDB.RPop(ctx, RedisCDNList).Result()
	if err != nil {
		return task, err
	}

	err = json.UnmarshalFromString(value, &task)
	return task, err
}

// 自增注册人数
func IncrRegCount(now time.Time) (int64, error) {
	return RDB.Incr(ctx, fmt.Sprintf(RedisUserRegCount, now.Format("20060102"))).Result()
}

// 删除注册人数
func DelRegCount(day string) (int64, error) {
	return RDB.Del(ctx, fmt.Sprintf(RedisUserRegCount, day)).Result()
}

// 自增写真任务数
func IncrPhotoCount(now time.Time, field string) (int64, error) {
	return RDB.HIncrBy(ctx, fmt.Sprintf(RedisUserPhotoCountHash, now.Format("20060102")), field, 1).Result()
}

// 删除写真任务数
func DelPhotoCount(day string) error {
	return RDB.Del(ctx, fmt.Sprintf(RedisUserPhotoCountHash, day)).Err()
}

// 增加模板使用者
func AddTemplateUser(templateId, userId int) (int64, error) {
	return RDB.SAdd(ctx, fmt.Sprintf(RedisTemplateSet, templateId), userId).Result()
}

// 锁定订单
func LockOrder(orderNum string) (bool, error) {
	return RDB.SetNX(ctx, fmt.Sprintf(RedisOrderLock, orderNum), 1, time.Second*60).Result()
}

// 解锁订单
func UnlockOrder(orderNum string) error {
	return RDB.Del(ctx, fmt.Sprintf(RedisOrderLock, orderNum)).Err()
}
