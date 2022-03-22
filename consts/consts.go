package consts

type TraceKey string

const (
	TimeFormatSecond = "2006-01-02 15:04:05"
	TimeFormatLog    = "2006-01-02T15"

	MethodPrecreate = "precreate"
	MethodUpload    = "upload"
	MethodCreate    = "create"

	AutoInitConstant = 1

	Size4MB  = 4 * 1024 * 1024
	Size16MB = 16 * 1024 * 1024
	Size32MB = 32 * 1024 * 1024

	ZipQuality50  = 50
	ZipQuality70  = 70
	ZipQuality100 = 100

	ModeManual          = 1 // 手动上传
	ModeBatch           = 2 // 批量上传
	ModeFileAutoBackup  = 3 // 文件自动备份
	ModeAlbumAutoBackup = 4 // 相册自动备份
	ModeVideoAutoBackup = 5 // 视频自动备份

	EnableMultiVersion  = 1 // 开启多版本支持
	DisableMultiVersion = 0 // 不开启多版本支持

	CategoryVideo       = 1 // 视频
	CategoryAudio       = 2 // 音频
	CategoryPhoto       = 3 // 图片
	CategoryDocument    = 4 // 文档
	CategoryApplication = 5 // 应用
	CategoryOthers      = 7 // 其他
	CategorySeed        = 8 // 种子

	RTypeError           = 0 // 如果存在同名文件返回错误
	RTypeRename          = 1 // 如果存在同名文件进行重命名
	RTypeBlockListRename = 2 // 如果存在同名文件且blockList不同是进行重命名
	RTypeOverride        = 3 // 如果存在同名文件进行覆盖

	ErrnoSuccess            = 0  // 返回成功的错误码
	ErrnoAccessTokenInvalid = -6 // access_token失效的错误吗

	MaxRetryCount = 3 // 最大上传次数
)

const (
	AuthorizationCodeUrl  = `https://openapi.baidu.com/oauth/2.0/authorize?response_type=code&client_id=%s&redirect_uri=oob&scope=netdisk&display=popup`
	AccessTokenCodeUrl    = `https://openapi.baidu.com/oauth/2.0/token?grant_type=authorization_code&code=%s&client_id=%s&client_secret=%s&redirect_uri=oob`
	AccessTokenRefreshUrl = `https://openapi.baidu.com/oauth/2.0/token?grant_type=refresh_token&refresh_token=%s&client_id=%s&client_secret=%s&scope=netdisk`
)

// 日志相关信息
const (
	LogTraceKey   = TraceKey("trace_id")
	LogTimeLayout = "2006-01-02 15:04:05.000-07:00"

	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

// 文件上传状态
const (
	UploadStatusNoUploaded = 0 // 未上传
	UploadStatusUploading  = 1 // 上传中
	UploadStatusUploaded   = 2 // 上传完成
	UploadStatusFail       = 3 // 上传失败
)
