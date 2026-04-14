package dao

import (
	"errors"

	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
)

type BootFuncInterface interface {
	Boot(opt model.Opts) error
}

type BootFunc func(opts model.Opts) error

type dSetting struct {
	optBoots map[string]BootFunc
}

var Setting = &dSetting{
	optBoots: map[string]BootFunc{},
}

func (o *dSetting) OptRegister(name string, defaultOpts model.Opts, bf BootFunc) {
	if !Ready() {
		return // 如果数据库还没装好则先跳过
	}

	opts, err := o.Get(name)
	if err != nil {
		err = o.set(name, defaultOpts)
		if err != nil {
			log.Error().Err(err).Send()
		}
		return
	}

	updated := false
	//默认值中存在，配置中没有，加入配置中
	for k, v := range defaultOpts {
		if !utils.CheckKeyInMap(k, opts) {
			opts[k] = v
			updated = true
		}
	}
	if updated {
		err = o.set(name, opts)
		if err != nil {
			log.Error().Err(err).Send()
		}
	}

	// 检查boot参数是否存在
	// 如果不存在则直接跳过
	if len(opts) == 0 {
		log.Warn().Str("options", name).Msg("skip boot for the component")
		return
	}

	if bf == nil {
		return
	}

	// 如果存在则执行一次BootFunc
	o.optBoots[name] = bf
	if err := bf(opts); err != nil {
		log.Error().Err(err).Str("options", name).Msg("boot failed")
		return
	}
}

func (o *dSetting) Get(name string) (model.Opts, error) {
	if !Ready() {
		return nil, errors.New("ErrSystemNotInited")
	}
	ret := new(model.Setting)
	if err := gdb.First(ret, "name=?", name).Error; err != nil {
		return nil, err
	}
	return ret.Opts, nil
}

// GetValue 获取配置对象 value 传递对象指针
func (o *dSetting) GetValue(name string, value interface{}) error {
	if !Ready() {
		return errors.New("ErrSystemNotInited")
	}
	ret := new(model.Setting)
	if err := gdb.First(ret, "name=?", name).Error; err != nil {
		return err
	}
	return mapstructure.Decode(ret.Opts, value)
}

// Set 设置配置
func (o *dSetting) Set(name string, p model.Opts) error {
	if boot, ok := o.optBoots[name]; ok && boot != nil {
		if err := boot(p); err != nil {
			return err
		}
	}
	return o.set(name, p)
}

func (o *dSetting) set(name string, opts model.Opts) error {
	mOpt := &model.Setting{Name: name}
	gdb.First(mOpt, "name=?", name)
	if opts != nil {
		mOpt.Opts = opts
	}
	return gdb.Save(mOpt).Error
}

func (o *dSetting) FindSettings() (list []model.Setting, err error) {
	err = gdb.Find(&list).Error
	return list, err
}
