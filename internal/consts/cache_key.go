package consts

const (
	appname = "backup"
)

//Key 获取缓存key
func Key(funcn string, parms ...string) (res string) {
	res = appname + "_" + funcn
	for _, parm := range parms {
		res = res + "_" + parm
	}
	return
}
