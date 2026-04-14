package buserr

const (
	ErrWordPreview = "ErrWordPreview"
)

func ErrFuncInvalid(key string) BusinessMultiError {
	return Wrap(New("ErrFuncInvalid"), New(key))
}
func ErrFuncUserOverflow(number string) BusinessError {
	return NewWithDetail("ErrFuncUserOverflow", number)
}

func ErrFuncInvalidParams(key string) BusinessMultiError {
	return Wrap(New("ErrFuncInvalidParams"), New(key))
}

func ErrFuncNotExist(key string) BusinessMultiError {
	return Wrap(New("ErrFuncNotExist"), New(key))
}
func ErrFuncExist(key string) BusinessMultiError {
	return Wrap(New("ErrFuncExist"), New(key))
}

func ErrFuncNotSupport(key string) BusinessMultiError {
	return Wrap(New("ErrFuncNotSupport"), New(key))
}

func ErrFuncMustBeNotNull(key string) BusinessMultiError {
	return Wrap(New("ErrFuncMustBeNotNull"), New(key))
}
func ErrFuncNotConfig(key string) BusinessMultiError {
	return Wrap(New("ErrFuncNotConfig"), New(key))
}

// ErrFuncOptFailed 操作失败
func ErrFuncOptFailed(errs ...string) BusinessMultiError {
	ls := make([]BusinessError, len(errs))
	for i, e := range errs {
		ls[i] = New(e)
	}
	return Wrap(New("ErrFuncOptFailed"), ls...)
}

func ErrFuncRequestFailed(key string) BusinessMultiError {
	return Wrap(New("ErrFuncRequestFailed"), New(key))
}

var (
	ErrInvalidID             = ErrFuncInvalid("ID")            //  无效的ID
	ErrUserDisabled          = New("ErrUserDisabled")          //  用户已禁用
	ErrUserApproving         = New("ErrUserApproving")         //  用户审核中
	ErrUserNotLogin          = New("ErrUserNotLogin")          //  用户未登录
	ErrInitialPassword       = New("ErrInitialPassword")       //  原密码错误
	ErrOrgNumberTooLarge     = New("ErrOrgNumberTooLarge")     // "组织部门数量过大"
	ErrInternalPrivateUser   = New("ErrInternalPrivateUser")   // "不能操作内置用户"
	ErrMaxOnlineTimeTooLarge = New("ErrMaxOnlineTimeTooLarge") // "最大在线时长过大"

	ErrMaxFilesNumTooLarge     = New("ErrMaxFilesNumTooLarge")     // "文件最大数过大"
	ErrWaterMarkContentTooLong = New("ErrWaterMarkContentTooLong") // "水印内容太长"
	ErrAdminCannotDisabled     = New("ErrAdminCannotDisabled")     // "管理员不能被禁用"

	ErrFileExist               = New("ErrFileExist")               //  文件已存在
	ErrFileCanNotRead          = New("ErrFileCanNotRead")          //  此文件不支持预览
	ErrInvalidMoveOperation    = New("ErrInvalidMoveOperation")    //   "无效的移动操作"
	ErrInvalidCopyOperation    = New("ErrInvalidCopyOperation")    //   "无效的复制操作"
	ErrIsFile                  = New("ErrIsFile")                  // "目标是个文件，不是文件夹"
	ErrIsDir                   = New("ErrIsDir")                   // "目标是个文件夹，不是文件"
	ErrFileNameTooLong         = New("ErrFileNameTooLong")         // "文件名太长"
	ErrFilePathTooLong         = New("ErrFilePathTooLong")         // "文件路径太长"
	ErrCreateFileOrDirectory   = New("ErrCreateFileOrDirectory")   // "无法创建文件或文件夹"
	ErrInvalidBucketName       = New("ErrInvalidBucketName")       // "无效的桶名"
	ErrInvalidFileType         = New("ErrInvalidFileType")         // "无效的文件类型"
	ErrInvalidFileName         = New("ErrInvalidFileName")         // "无效的文件名"
	ErrInvalidFilePath         = New("ErrInvalidFilePath")         // "无效的路径"
	ErrInvalidFileNameLength   = New("ErrInvalidFileNameLength")   // "无效的文件名长度"
	ErrInvalidMaxDownloadSpeed = New("ErrInvalidMaxDownloadSpeed") // "无效的最大下载速度"
	ErrInvalidMaxUploadSpeed   = New("ErrInvalidMaxUploadSpeed")   // "无效的最大上传速度"
	ErrOverOrgFileNumLimit     = New("ErrOverOrgFileNumLimit")     // "超过部门最大文件数量限制"
	ErrOverUserFileNumLimit    = New("ErrOverUserFileNumLimit")    // 超过用户最大文件数量限制
	ErrOverSystemFileNumLimit  = New("ErrOverSystemFileNumLimit")  // 超过系统最大文件数量限制
	ErrOverSystemFileSizeLimit = New("ErrOverSystemFileNumLimit")  // 超过系统最大文件大小限制

	ErrOverUserQuota     = New("ErrOverUserQuota")     // 超过用户配额限制
	ErrOverUserMaxSpace  = New("ErrOverUserMaxSpace")  // "配额超过了最大用户空间大小"
	ErrUserQuotaTooSmall = New("ErrUserQuotaTooSmall") // "配额不能小于用户已经占用空间大小"
	ErrOverOrgMaxSpace   = New("ErrOverOrgMaxSpace")   // "配额超过了部门最大空间大小"
	ErrOverGroupMaxSpace = New("ErrOverOrgMaxSpace")   // "配额超过了部门最大空间大小"

	ErrFileShareSwitchUnopened       = New("ErrFileShareSwitchUnopened")       //"外链分享功能未开启"
	ErrFileShareExpired              = New("ErrFileShareExpired")              //"分享的文件已过期"
	ErrInvalidFileShareSecret        = New("ErrInvalidFileShareSecret")        //"无效的分享密钥"
	ErrFileCantDownload              = New("ErrFileCantDownload")              //"文件禁止下载"
	ErrFileNotPermission             = New("ErrFileNotPermission")             //"文件禁止下载"
	ErrFileDownloadCountLimit        = New("ErrFileDownloadCountLimit")        // "文件下载已达上限"
	ErrFileDownloadConcurrentTooMuch = New("ErrFileDownloadConcurrentTooMuch") // "文件并发下载已达上限"

	ErrInvalidStatusCode = New("ErrInvalidStatusCode") //  无效的状态码
	ErrInvalidLicense    = New("ErrInvalidLicense")    //  无效的许可

	ErrSystemNotInited          = New("ErrSystemNotInited")          //  系统没有初始化
	ErrImportUserNumberTooLarge = New("ErrImportUserNumberTooLarge") //  "导入用户数量过多"
	ErrSystemRegistryClose      = New("ErrSystemRegistryClose")      //"系统不支持注册用户"

	ErrCmdTimeout                  = New("ErrCmdTimeout")
	ErrDefaultUserQuota            = New("ErrDefaultUserQuota")
	ErrDefaultOrgQuota             = New("ErrDefaultOrgQuota")
	ErrStorageDeviceInUse          = New("ErrStorageDeviceInUse")
	ErrInvalidSignature            = New("ErrInvalidSignature")
	ErrDefaultGroupQuota           = New("ErrDefaultGroupQuota")
	ErrIsInternalPrivatePermission = New("ErrIsInternalPrivatePermission")
	ErrConnectLDAPServerFailed     = New("ErrConnectLDAPServerFailed")
	ErrRecordNotFound              = New("ErrRecordNotFound")
	ErrInvalidIP                   = New("ErrInvalidIP")
	ErrNotImplemented              = New("ErrNotImplemented")

	ErrStatusRejected  = New("ErrStatusRejectedNoPermission")
	ErrStatusCanceled  = New("ErrStatusCanceled")
	ErrStatusApproving = New("ErrStatusApproving")
	ErrStatusApproved  = New("ErrStatusApproved")
)
