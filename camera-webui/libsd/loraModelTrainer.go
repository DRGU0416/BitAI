package libsd

import (
	"bytes"
	"camera-webui/logger"
	"fmt"
	"io"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	batchSize               int
	saveEveryNEpochs        int
	cpuThreadsPerCore       int
	networkRank             int
	optimizer               string
	useXFormers             bool
	gradientAccumulateSteps int
	priorLossWeight         int
	Loop                    int

	logTask = logger.New("logs/params.log")
)

func init() {
	initConfig()
	go dynamicConfig()
}

func initConfig() {
	batchSize = viper.GetInt("webui.batch_size")
	saveEveryNEpochs = viper.GetInt("webui.save_every_n_epochs")
	cpuThreadsPerCore = viper.GetInt("webui.cpu_threads_per_core")
	networkRank = viper.GetInt("webui.network_rank")
	optimizer = viper.GetString("webui.optimizer")
	useXFormers = viper.GetBool("webui.use_xformers")
	gradientAccumulateSteps = viper.GetInt("webui.gradient_accumulate_steps")
	priorLossWeight = viper.GetInt("webui.prior_loss_weight")
	Loop = viper.GetInt("webui.loop")
}

func dynamicConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		initConfig()
	})
}

// ========== SD(v1.3.1)接口 ================
type SDLoraModelTrainer struct {
	// 常用配置
	BaseModel           string  `json:"baseModel"`
	ModelQuickPick      string  `json:"modelQuickPick"`
	IsV2                bool    `json:"isV2"`
	ImageFolder         string  `json:"imageFolder"`
	OutputFolder        string  `json:"outputFolder"`
	LoggingFolder       string  `json:"loggingFolder"`
	ModelOutputName     string  `json:"modelOutputName"`
	BatchSize           int     `json:"batchSize"`
	Epoch               int     `json:"epoch"`
	SaveEveryNEpochs    int     `json:"saveEveryNEpochs"`
	CpuThreadsPerCore   int     `json:"cpuThreadsPerCore"`
	Seed                float64 `json:"seed,omitempty"`
	NetworkRank         int     `json:"networkRank"`
	MaxResolutionWidth  int     `json:"maxResolutionWidth"`
	MaxResolutionHeight int     `json:"maxResolutionHeight"`
	Optimizer           string  `json:"optimizer"`
	UseXFormers         bool    `json:"useXFormers"`

	// 不常用默认配置
	SaveTrainedModelAs           string  `json:"saveTrainedModelAs"`
	VParameterization            bool    `json:"vParameterization"`
	RegularisationFolder         string  `json:"regularisationFolder"`
	TrainingComment              string  `json:"trainingComment"`
	LoraType                     string  `json:"loraType"`
	LoraNetworkWeights           string  `json:"loraNetworkWeights"`
	CaptionExtension             string  `json:"captionExtension"`
	MixedPrecision               string  `json:"mixedPrecision"`
	SavePrecision                string  `json:"savePrecision"`
	LearningRate                 float64 `json:"learningRate"`
	LrScheduler                  string  `json:"lrScheduler"`
	PercentOfStepsForLRWarmup    int     `json:"percentOfStepsForLRWarmup"`
	OptimizerExtraArguments      string  `json:"optimizerExtraArguments"`
	TextEncoderLearningRate      float64 `json:"textEncoderLearningRate"`
	UnetLearningRate             float64 `json:"unetLearningRate"`
	NetworkAlpha                 int     `json:"networkAlpha"`
	StopTextEncoderTraining      int     `json:"stopTextEncoderTraining"`
	EnableBuckets                bool    `json:"enableBuckets"`
	NoTokenPadding               bool    `json:"noTokenPadding"`
	GradientAccumulateSteps      int     `json:"gradientAccumulateSteps"`
	PriorLossWeight              int     `json:"priorLossWeight"`
	LrNumberOfCycles             int     `json:"lrNumberOfCycles,omitempty"`
	LrPower                      int     `json:"lrPower,omitempty"`
	AdditionalParameters         string  `json:"additionalParameters"`
	KeepNTokens                  string  `json:"keepNTokens"`
	ClipSkip                     int     `json:"clipSkip"`
	MaxTokenLength               int     `json:"maxTokenLength"`
	FullFP16Training             bool    `json:"fullFP16Training"`
	GradientCheckPointing        bool    `json:"gradientCheckPointing"`
	ShuffleCaption               bool    `json:"shuffleCaption"`
	PersistentDataLoader         bool    `json:"persistentDataLoader"`
	MemoryEfficientAttention     bool    `json:"memoryEfficientAttention"`
	ColorAugmentation            bool    `json:"colorAugmentation"`
	FlipAugmentation             bool    `json:"flipAugmentation"`
	DoNotUpscaleBucketResolution bool    `json:"doNotUpscaleBucketResolution"`
	BucketResolutionSteps        int     `json:"bucketResolutionSteps"`
	RandomCropInteadOfCenterCrop bool    `json:"randomCropInsteadOfCenterCrop"`
	NoiseOffset                  float64 `json:"noiseOffset,omitempty"`
	DropoutCaptionEveryNEpochs   int     `json:"dropoutCaptionEveryNEpochs"`
	RateOfCaptionDropout         float64 `json:"rateOfCaptionDropout"`
	SaveTrainingState            bool    `json:"saveTrainingState"`
	ResumeFromSavedTrainingState string  `json:"resumeFromSavedTrainingState"`
	MaxTrainEpoch                int     `json:"maxTrainEpoch,omitempty"`
	MaxNumOfWorkersForDataLoader int     `json:"maxNumOfWorkersForDataLoader,omitempty"`
	SampleEveryNEpochs           int     `json:"sampleEveryNEpochs"`
	SampleEveryNSteps            int     `json:"sampleEveryNSteps"`
	SampleSampler                string  `json:"sampleSampler"`
	SamplePrompts                string  `json:"samplePrompts"`

	SaveEveryNSteps         int     `json:"saveEveryNSteps"`
	SaveLastNModels         int     `json:"saveLastNModels"`
	SaveLastNStates         int     `json:"saveLastNStates"`
	MinSNRGamma             int     `json:"minSNRGamma"`
	VaeBatchSize            int     `json:"vaeBatchSize"`
	CacheLatent             bool    `json:"cacheLatent"`
	CacheLatentToDisk       bool    `json:"cacheLatentToDisk"`
	MultiResNoiseIterations int     `json:"multiResNoiseIterations"`
	MultiResNoiseDiscount   float64 `json:"multiResNoiseDiscount"`
	ConvolutionRank         int     `json:"convolutionRank"`
	ConvolutionAlpha        float64 `json:"convolutionAlpha"`
	DownLRWeights           string  `json:"downLRWeights"`
	MidLRWeights            string  `json:"midLRWeights"`
	UpLRWeights             string  `json:"upLRWeights"`
	BlocksLRThreshold       string  `json:"blocksLRThreshold"`
	BlockDims               string  `json:"blockDims"`
	BlockAlphas             string  `json:"blockAlphas"`
	ConvDims                string  `json:"convDims"`
	ConvAlphas              string  `json:"convAlphas"`
	WeightedCaptions        bool    `json:"weightedCaptions"`
	DyLoRAUnit              int     `json:"dyLoRAUnit"`
}

type SDModelTrainResult struct {
	Data            []any   `json:"data"`
	IsGenerating    bool    `json:"is_generating"`
	Duration        float64 `json:"duration"`
	AverageDuration float64 `json:"average_duration"`
}

func (t SDLoraModelTrainer) toJsonData() []any {
	data := make([]any, 86)
	label := make(map[string]string)
	label["label"] = "False"
	data[0] = label
	data[1] = t.BaseModel
	data[2] = t.IsV2
	data[3] = t.VParameterization
	data[4] = t.LoggingFolder
	data[5] = t.ImageFolder
	data[6] = t.RegularisationFolder
	data[7] = t.OutputFolder
	data[8] = fmt.Sprintf("%d,%d", t.MaxResolutionWidth, t.MaxResolutionHeight)
	data[9] = fmt.Sprintf("%f", t.LearningRate)
	data[10] = t.LrScheduler
	data[11] = t.PercentOfStepsForLRWarmup
	data[12] = t.BatchSize
	data[13] = t.Epoch
	data[14] = t.SaveEveryNEpochs
	data[15] = t.MixedPrecision
	data[16] = t.SavePrecision
	data[17] = "" //this.seed.HasValue?this.seed.ToString():"",
	data[18] = t.CpuThreadsPerCore
	data[19] = t.CacheLatent
	data[20] = t.CacheLatentToDisk
	data[21] = t.CaptionExtension
	data[22] = t.EnableBuckets
	data[23] = t.GradientCheckPointing
	data[24] = t.FullFP16Training
	data[25] = t.NoTokenPadding
	data[26] = t.StopTextEncoderTraining
	data[27] = t.UseXFormers
	data[28] = t.SaveTrainedModelAs
	data[29] = t.ShuffleCaption
	data[30] = t.SaveTrainingState
	data[31] = t.ResumeFromSavedTrainingState
	data[32] = t.PriorLossWeight
	data[33] = fmt.Sprintf("%f", t.TextEncoderLearningRate)
	data[34] = fmt.Sprintf("%f", t.UnetLearningRate)
	data[35] = t.NetworkRank
	data[36] = t.LoraNetworkWeights
	data[37] = t.ColorAugmentation
	data[38] = t.FlipAugmentation
	data[39] = t.ClipSkip
	data[40] = t.GradientAccumulateSteps
	data[41] = t.MemoryEfficientAttention
	data[42] = t.ModelOutputName
	data[43] = t.ModelQuickPick
	data[44] = fmt.Sprintf("%d", t.MaxTokenLength)
	data[45] = ""  // this.maxTrainEpoch.HasValue? this.maxTrainEpoch.ToString() : "",
	data[46] = "0" // this.maxNumOfWorkersForDataLoader.HasValue? this.maxNumOfWorkersForDataLoader.ToString() : "",
	data[47] = t.NetworkAlpha
	data[48] = t.TrainingComment
	data[49] = t.KeepNTokens
	data[50] = "" //this.lrNumberOfCycles.HasValue? this.lrNumberOfCycles.Value.ToString() : "",
	data[51] = "" //this.lrPower.HasValue? this.lrPower.Value.ToString() : "",
	data[52] = t.PersistentDataLoader
	data[53] = t.DoNotUpscaleBucketResolution
	data[54] = t.RandomCropInteadOfCenterCrop
	data[55] = t.BucketResolutionSteps
	data[56] = t.DropoutCaptionEveryNEpochs
	data[57] = t.RateOfCaptionDropout
	data[58] = t.Optimizer
	data[59] = t.OptimizerExtraArguments
	data[60] = 0 //this.noiseOffset.HasValue?this.noiseOffset.ToString():"",
	data[61] = t.MultiResNoiseIterations
	data[62] = t.MultiResNoiseDiscount
	data[63] = t.LoraType
	data[64] = t.ConvolutionRank
	data[65] = t.ConvolutionAlpha
	data[66] = t.SampleEveryNSteps
	data[67] = t.SampleEveryNEpochs
	data[68] = t.SampleSampler
	data[69] = t.SamplePrompts
	data[70] = t.AdditionalParameters
	data[71] = t.VaeBatchSize
	data[72] = t.MinSNRGamma
	data[73] = t.DownLRWeights
	data[74] = t.MidLRWeights
	data[75] = t.UpLRWeights
	data[76] = t.BlocksLRThreshold
	data[77] = t.BlockDims
	data[78] = t.BlockAlphas
	data[79] = t.ConvDims
	data[80] = t.ConvAlphas
	data[81] = t.WeightedCaptions
	data[82] = t.DyLoRAUnit
	data[83] = t.SaveEveryNSteps
	data[84] = t.SaveLastNModels
	data[85] = t.SaveLastNStates

	return data
}

func (t SDLoraModelTrainer) TrainLORAModel(session string, imgCnt int) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logTask.Errorf("recover success. err: %s", e)
			err = fmt.Errorf("TrainLORAModel error = %s", e)
		}
	}()
	//常用配置
	t.BatchSize = batchSize
	// x := 45 * math.Pow(float64(batchSize), float64(2))
	// y := float64(imgCnt * Loop)
	// t.Epoch = int(math.Floor(x / y))
	t.Epoch = 35
	t.ModelQuickPick = "custom"
	t.SaveEveryNEpochs = saveEveryNEpochs
	t.SaveLastNModels = 1
	t.CpuThreadsPerCore = cpuThreadsPerCore
	t.NetworkRank = networkRank
	t.Optimizer = optimizer
	t.UseXFormers = useXFormers
	t.GradientAccumulateSteps = gradientAccumulateSteps
	t.PriorLossWeight = priorLossWeight
	t.MaxResolutionWidth = 512
	t.MaxResolutionHeight = 512
	if t.IsV2 {
		t.MaxResolutionWidth = 768
		t.MaxResolutionHeight = 768
	}

	//不常用默认配置
	t.SaveTrainedModelAs = "safetensors"
	t.VParameterization = false
	t.RegularisationFolder = "" //有可能会用到
	t.CacheLatent = true
	t.LoraType = "Standard"
	t.MixedPrecision = "bf16"
	t.SavePrecision = "bf16"
	t.LearningRate = 0.0001
	t.LrScheduler = "cosine_with_restarts"
	t.PercentOfStepsForLRWarmup = 0
	t.TextEncoderLearningRate = 1e-5
	t.UnetLearningRate = 0.0001
	t.NetworkAlpha = 32

	//以下四项可能影响速度
	t.KeepNTokens = "0"
	t.ConvolutionRank = 32
	t.ConvolutionAlpha = 32
	t.DyLoRAUnit = 1

	t.EnableBuckets = false
	t.ClipSkip = 1
	t.MaxTokenLength = 75
	t.FlipAugmentation = false //改成true
	t.DoNotUpscaleBucketResolution = true
	t.BucketResolutionSteps = 64
	t.SampleSampler = "euler_a"

	// 组装结构体
	body := make(map[string]any)
	body["fn_index"] = 79
	body["data"] = t.toJsonData()
	body["session_hash"] = session

	params, _ := json.MarshalToString(t)
	logTask.Infof("train params: %s", params)

	url := fmt.Sprintf("%s/run/predict", kohyaHost)

	byteMsg, err := json.Marshal(body)
	if err != nil {
		return err
	}
	logTask.Infof("train input param: %s", string(byteMsg))

	request, _ := http.NewRequest("POST", url, bytes.NewReader(byteMsg))
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := SDModelTrainResult{}
	err = json.Unmarshal(result, &data)
	return err
}

// ========== SD(v1.6.0)接口 ================
type SDLoraModelTrainer160 struct {
	// 常用配置
	BaseModel           string `json:"baseModel"`
	ModelQuickPick      string `json:"modelQuickPick"`
	IsV2                bool   `json:"isV2"`
	ImageFolder         string `json:"imageFolder"`
	OutputFolder        string `json:"outputFolder"`
	LoggingFolder       string `json:"loggingFolder"`
	ModelOutputName     string `json:"modelOutputName"`
	BatchSize           int    `json:"batchSize"`
	Epoch               int    `json:"epoch"`
	SaveEveryNEpochs    int    `json:"saveEveryNEpochs"`
	CpuThreadsPerCore   int    `json:"cpuThreadsPerCore"`
	Seed                *int64 `json:"seed,omitempty"`
	NetworkRank         int    `json:"networkRank"`
	MaxResolutionWidth  int    `json:"maxResolutionWidth"`
	MaxResolutionHeight int    `json:"maxResolutionHeight"`
	Optimizer           string `json:"optimizer"`

	// 不常用默认配置
	SaveTrainedModelAs           string  `json:"saveTrainedModelAs"`
	VParameterization            bool    `json:"vParameterization"`
	RegularisationFolder         string  `json:"regularisationFolder"`
	TrainingComment              string  `json:"trainingComment"`
	LoraType                     string  `json:"loraType"`
	LoraNetworkWeights           string  `json:"loraNetworkWeights"`
	CaptionExtension             string  `json:"captionExtension"`
	MixedPrecision               string  `json:"mixedPrecision"`
	SavePrecision                string  `json:"savePrecision"`
	LearningRate                 float64 `json:"learningRate"`
	LrScheduler                  string  `json:"lrScheduler"`
	PercentOfStepsForLRWarmup    int     `json:"percentOfStepsForLRWarmup"`
	OptimizerExtraArguments      string  `json:"optimizerExtraArguments"`
	TextEncoderLearningRate      float64 `json:"textEncoderLearningRate"`
	UnetLearningRate             float64 `json:"unetLearningRate"`
	NetworkAlpha                 int     `json:"networkAlpha"`
	StopTextEncoderTraining      int     `json:"stopTextEncoderTraining"`
	EnableBuckets                bool    `json:"enableBuckets"`
	NoTokenPadding               bool    `json:"noTokenPadding"`
	GradientAccumulateSteps      int     `json:"gradientAccumulateSteps"`
	PriorLossWeight              int     `json:"priorLossWeight"`
	LrNumberOfCycles             int     `json:"lrNumberOfCycles,omitempty"`
	LrPower                      *int    `json:"lrPower,omitempty"`
	AdditionalParameters         string  `json:"additionalParameters"`
	KeepNTokens                  int     `json:"keepNTokens"`
	ClipSkip                     int     `json:"clipSkip"`
	MaxTokenLength               int     `json:"maxTokenLength"`
	FullFP16Training             bool    `json:"fullFP16Training"`
	GradientCheckPointing        bool    `json:"gradientCheckPointing"`
	ShuffleCaption               bool    `json:"shuffleCaption"`
	PersistentDataLoader         bool    `json:"persistentDataLoader"`
	MemoryEfficientAttention     bool    `json:"memoryEfficientAttention"`
	ColorAugmentation            bool    `json:"colorAugmentation"`
	FlipAugmentation             bool    `json:"flipAugmentation"`
	DoNotUpscaleBucketResolution bool    `json:"doNotUpscaleBucketResolution"`
	BucketResolutionSteps        int     `json:"bucketResolutionSteps"`
	RandomCropInteadOfCenterCrop bool    `json:"randomCropInsteadOfCenterCrop"`
	NoiseOffset                  float64 `json:"noiseOffset,omitempty"`
	DropoutCaptionEveryNEpochs   int     `json:"dropoutCaptionEveryNEpochs"`
	RateOfCaptionDropout         float64 `json:"rateOfCaptionDropout"`
	SaveTrainingState            bool    `json:"saveTrainingState"`
	ResumeFromSavedTrainingState string  `json:"resumeFromSavedTrainingState"`
	MaxTrainEpoch                *int    `json:"maxTrainEpoch,omitempty"`
	MaxNumOfWorkersForDataLoader *int    `json:"maxNumOfWorkersForDataLoader,omitempty"`
	SampleEveryNEpochs           int     `json:"sampleEveryNEpochs"`
	SampleEveryNSteps            int     `json:"sampleEveryNSteps"`
	SampleSampler                string  `json:"sampleSampler"`
	SamplePrompts                string  `json:"samplePrompts"`

	SaveEveryNSteps         int     `json:"saveEveryNSteps"`
	SaveLastNModels         int     `json:"saveLastNModels"`
	SaveLastNStates         int     `json:"saveLastNStates"`
	MinSNRGamma             int     `json:"minSNRGamma"`
	VaeBatchSize            int     `json:"vaeBatchSize"`
	CacheLatent             bool    `json:"cacheLatent"`
	CacheLatentToDisk       bool    `json:"cacheLatentToDisk"`
	MultiResNoiseIterations int     `json:"multiResNoiseIterations"`
	MultiResNoiseDiscount   float64 `json:"multiResNoiseDiscount"`
	ConvolutionRank         int     `json:"convolutionRank"`
	ConvolutionAlpha        float64 `json:"convolutionAlpha"`
	DownLRWeights           string  `json:"downLRWeights"`
	MidLRWeights            string  `json:"midLRWeights"`
	UpLRWeights             string  `json:"upLRWeights"`
	BlocksLRThreshold       string  `json:"blocksLRThreshold"`
	BlockDims               string  `json:"blockDims"`
	BlockAlphas             string  `json:"blockAlphas"`
	ConvDims                string  `json:"convDims"`
	ConvAlphas              string  `json:"convAlphas"`
	WeightedCaptions        bool    `json:"weightedCaptions"`
	DyLoRAUnit              int     `json:"dyLoRAUnit"`

	// 新增加的字段
	IsSDXLModel               bool    `json:"isSDXLModel"`
	MinBucketResolution       int     `json:"minBucketResolution"`
	MaxBucketResolution       int     `json:"maxBucketResolution"`
	CrossAttention            string  `json:"crossAttention"`
	DimFromWeights            bool    `json:"dimFromWeights"`
	MaxTrainSteps             *int    `json:"maxTrainSteps"`
	VPredLikeLoss             float64 `json:"vPredLikeLoss"`
	NoiseOffsetType           string  `json:"noiseOffsetType"`
	AdaptiveNoiseScale        float64 `json:"adaptiveNoiseScale"`
	UseCPDecomposition        bool    `json:"useCPDecomposition"`
	LoKrDecomposeBoth         bool    `json:"loKrDecomposeBoth"`
	IA3TrainOnInput           bool    `json:"iA3TrainOnInput"`
	LoKrFactor                float64 `json:"loKrFactor"`
	WanDBLogging              bool    `json:"wanDBLogging"`
	WanDBAPIKey               string  `json:"wanDBAPIKey"`
	ScaleVPredictionLoss      bool    `json:"scaleVPredictionLoss"`
	ScaleWeightNorms          float64 `json:"scaleWeightNorms"`
	NetworkDropout            float64 `json:"networkDropout"`
	RankDropout               float64 `json:"rankDropout"`
	ModuleDropout             float64 `json:"moduleDropout"`
	CacheTextEncoderOutputs   bool    `json:"cacheTextEncoderOutputs"`
	NoHalfVAE                 bool    `json:"noHalfVAE"`
	FullBF16Training          bool    `json:"fullBF16Training"`
	MinTimestep               int     `json:"minTimestep"`
	MaxTimestep               int     `json:"maxTimestep"`
	LrSchedulerExtraArguments string  `json:"lrSchedulerExtraArguments"`
}

func (t SDLoraModelTrainer160) toJsonData() []any {
	data := make([]any, 112)
	label := make(map[string]string)
	label["label"] = "False"
	data[0] = label
	data[1] = label
	data[2] = t.BaseModel
	data[3] = t.IsV2
	data[4] = t.VParameterization
	data[5] = t.IsSDXLModel
	data[6] = t.LoggingFolder
	data[7] = t.ImageFolder
	data[8] = t.RegularisationFolder
	data[9] = t.OutputFolder
	data[10] = fmt.Sprintf("%d,%d", t.MaxResolutionWidth, t.MaxResolutionHeight)
	data[11] = fmt.Sprintf("%f", t.LearningRate)
	data[12] = t.LrScheduler
	data[13] = t.PercentOfStepsForLRWarmup
	data[14] = t.BatchSize
	data[15] = t.Epoch
	data[16] = t.SaveEveryNEpochs
	data[17] = t.MixedPrecision
	data[18] = t.SavePrecision
	if t.Seed != nil {
		data[19] = fmt.Sprintf("%d", *t.Seed)
	} else {
		data[19] = ""
	}
	data[20] = t.CpuThreadsPerCore
	data[21] = t.CacheLatent
	data[22] = t.CacheLatentToDisk
	data[23] = t.CaptionExtension
	data[24] = t.EnableBuckets
	data[25] = t.GradientCheckPointing
	data[26] = t.FullFP16Training
	data[27] = t.NoTokenPadding
	data[28] = t.StopTextEncoderTraining
	data[29] = t.MinBucketResolution
	data[30] = t.MaxBucketResolution
	data[31] = t.CrossAttention
	data[32] = t.SaveTrainedModelAs
	data[33] = t.ShuffleCaption
	data[34] = t.SaveTrainingState
	data[35] = t.ResumeFromSavedTrainingState
	data[36] = t.PriorLossWeight
	data[37] = t.TextEncoderLearningRate
	data[38] = t.UnetLearningRate
	data[39] = t.NetworkRank
	data[40] = t.LoraNetworkWeights
	data[41] = t.DimFromWeights
	data[42] = t.ColorAugmentation
	data[43] = t.FlipAugmentation
	data[44] = fmt.Sprintf("%d", t.ClipSkip)
	data[45] = t.GradientAccumulateSteps
	data[46] = t.MemoryEfficientAttention
	data[47] = t.ModelOutputName
	data[48] = t.ModelQuickPick
	data[49] = fmt.Sprintf("%d", t.MaxTokenLength)
	if t.MaxTrainEpoch != nil {
		data[50] = fmt.Sprintf("%d", *t.MaxTrainEpoch)
	} else {
		data[50] = ""
	}
	if t.MaxTrainSteps != nil {
		data[51] = fmt.Sprintf("%d", *t.MaxTrainSteps)
	} else {
		data[51] = ""
	}
	if t.MaxNumOfWorkersForDataLoader != nil {
		data[52] = fmt.Sprintf("%d", *t.MaxNumOfWorkersForDataLoader)
	} else {
		data[52] = ""
	}
	data[53] = t.NetworkAlpha
	data[54] = t.TrainingComment
	data[55] = fmt.Sprintf("%d", t.KeepNTokens)
	data[56] = fmt.Sprintf("%d", t.LrNumberOfCycles)
	if t.LrPower != nil {
		data[57] = fmt.Sprintf("%d", *t.LrPower)
	} else {
		data[57] = ""
	}
	data[58] = t.PersistentDataLoader
	data[59] = t.DoNotUpscaleBucketResolution
	data[60] = t.RandomCropInteadOfCenterCrop
	data[61] = t.BucketResolutionSteps
	data[62] = t.VPredLikeLoss
	data[63] = t.DropoutCaptionEveryNEpochs
	data[64] = t.RateOfCaptionDropout
	data[65] = t.Optimizer
	data[66] = t.OptimizerExtraArguments
	data[67] = t.LrSchedulerExtraArguments
	data[68] = t.NoiseOffsetType
	data[69] = t.NoiseOffset
	data[70] = t.AdaptiveNoiseScale
	data[71] = t.MultiResNoiseIterations
	data[72] = t.MultiResNoiseDiscount
	data[73] = t.LoraType
	data[74] = t.LoKrFactor
	data[75] = t.UseCPDecomposition
	data[76] = t.LoKrDecomposeBoth
	data[77] = t.IA3TrainOnInput
	data[78] = t.ConvolutionRank
	data[79] = t.ConvolutionAlpha
	data[80] = t.SampleEveryNSteps
	data[81] = t.SampleEveryNEpochs
	data[82] = t.SampleSampler
	data[83] = t.SamplePrompts
	data[84] = t.AdditionalParameters
	data[85] = t.VaeBatchSize
	data[86] = t.MinSNRGamma
	data[87] = t.DownLRWeights
	data[88] = t.MidLRWeights
	data[89] = t.UpLRWeights
	data[90] = t.BlocksLRThreshold
	data[91] = t.BlockDims
	data[92] = t.BlockAlphas
	data[93] = t.ConvDims
	data[94] = t.ConvAlphas
	data[95] = t.WeightedCaptions
	data[96] = t.DyLoRAUnit
	data[97] = t.SaveEveryNSteps
	data[98] = t.SaveLastNModels
	data[99] = t.SaveLastNStates
	data[100] = t.WanDBLogging
	data[101] = t.WanDBAPIKey
	data[102] = t.ScaleVPredictionLoss
	data[103] = t.ScaleWeightNorms
	data[104] = t.NetworkDropout
	data[105] = t.RankDropout
	data[106] = t.ModuleDropout
	data[107] = t.CacheTextEncoderOutputs
	data[108] = t.NoHalfVAE
	data[109] = t.FullBF16Training
	data[110] = t.MinTimestep
	data[111] = t.MaxTimestep

	return data
}

func (t SDLoraModelTrainer160) TrainLORAModel(session string, imgCnt int) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logTask.Errorf("recover success. err: %s", e)
			err = fmt.Errorf("TrainLORAModel error = %s", e)
		}
	}()
	//常用配置
	t.BatchSize = batchSize
	// x := 45 * math.Pow(float64(batchSize), float64(2))
	// y := float64(imgCnt * Loop)
	// t.Epoch = int(math.Floor(x / y))
	t.Epoch = 35
	t.ModelQuickPick = "custom"
	t.SaveEveryNEpochs = saveEveryNEpochs
	t.SaveLastNModels = 1
	t.CpuThreadsPerCore = cpuThreadsPerCore
	t.NetworkRank = networkRank
	t.Optimizer = optimizer
	t.GradientAccumulateSteps = gradientAccumulateSteps
	t.PriorLossWeight = priorLossWeight
	t.MaxResolutionWidth = 512
	t.MaxResolutionHeight = 512
	if t.IsV2 {
		t.MaxResolutionWidth = 768
		t.MaxResolutionHeight = 768
	}

	//不常用默认配置
	t.SaveTrainedModelAs = "safetensors"
	t.VParameterization = false
	t.RegularisationFolder = "" //有可能会用到
	t.CacheLatent = true
	t.LoraType = "Standard"
	t.MixedPrecision = "bf16"
	t.SavePrecision = "bf16"
	t.LearningRate = 0.0001
	t.LrScheduler = "cosine_with_restarts"
	t.PercentOfStepsForLRWarmup = 0
	t.TextEncoderLearningRate = 1e-5
	t.UnetLearningRate = 0.0001
	t.NetworkAlpha = 32

	//以下四项可能影响速度
	t.KeepNTokens = 0
	t.ConvolutionRank = 1
	t.ConvolutionAlpha = 1
	t.DyLoRAUnit = 1

	t.EnableBuckets = false
	t.ClipSkip = 1
	t.MaxTokenLength = 75
	t.FlipAugmentation = false //改成true
	t.DoNotUpscaleBucketResolution = true
	t.BucketResolutionSteps = 64
	t.SampleSampler = "ddim"
	t.LrNumberOfCycles = 1

	// 新增加字段
	t.MinBucketResolution = 256
	t.MaxBucketResolution = 2048
	t.CrossAttention = "xformers"
	t.NoiseOffsetType = "Original"
	t.IA3TrainOnInput = true
	t.LoKrFactor = -1
	t.NoHalfVAE = true
	t.MaxTimestep = 1000

	MaxNumOfWorkersForDataLoader := 0
	t.MaxNumOfWorkersForDataLoader = &MaxNumOfWorkersForDataLoader

	// 组装结构体
	body := make(map[string]any)
	body["fn_index"] = 71
	body["data"] = t.toJsonData()
	body["session_hash"] = session

	params, _ := json.MarshalToString(t)
	logTask.Infof("train params: %s", params)

	url := fmt.Sprintf("%s/run/predict", kohyaHost)

	byteMsg, err := json.Marshal(body)
	if err != nil {
		return err
	}
	logTask.Infof("train input param: %s", string(byteMsg))

	request, _ := http.NewRequest("POST", url, bytes.NewReader(byteMsg))
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := SDModelTrainResult{}
	err = json.Unmarshal(result, &data)
	return err
}

func CancelTrainTask(session string) error {
	// 组装结构体
	body := make(map[string]any)
	body["fn_index"] = 72
	body["data"] = make([]int, 0)
	body["session_hash"] = session

	url := fmt.Sprintf("%s/run/predict", kohyaHost)

	byteMsg, err := json.Marshal(body)
	if err != nil {
		return err
	}
	logTask.Infof("train cancel param: %s", string(byteMsg))
	request, _ := http.NewRequest("POST", url, bytes.NewReader(byteMsg))
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := SDModelTrainResult{}
	if err = json.Unmarshal(result, &data); err != nil {
		fmt.Printf("取消任务解析失败 %s", err)
	}

	return err
}
