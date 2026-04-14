package dao

import (
	"github.com/alayou/techstack/model"
)

type dPackage struct{}

var Lirary = new(dPackage)

func (s *dPackage) GetLibaryByPurl(purl string) (*model.Package, error) {
	var lv model.PackageVersion
	err := gdb.Where("purl = ?", purl).First(&lv).Error
	if err != nil {
		return nil, err
	}

	var lib model.Package
	err = gdb.Where("id = ?", lv.PackageID).First(&lib).Error
	if err != nil {
		return nil, err
	}
	return &lib, nil
}

func (s *dPackage) GetLibaryByPurlType(pty, name string) (*model.Package, error) {
	var lib model.Package
	err := gdb.Where("purl_type = ? AND name = ?", pty, name).First(&lib).Error
	if err != nil {
		return nil, err
	}
	return &lib, nil
}

func (s *dPackage) GetLibaryByID(id int64) (*model.Package, error) {
	var lib model.Package
	err := gdb.Where("id = ?", id).First(&lib).Error
	if err != nil {
		return nil, err
	}
	return &lib, nil
}

func (s *dPackage) GetVersions(id int64) (ls []*model.PackageVersion, err error) {
	err = gdb.Where("package_id = ?", id).Order("published_at desc").Find(&ls).Error
	return
}

func (s *dPackage) GetVersionByID(versionID int64) (out *model.PackageVersion, err error) {
	var lv model.PackageVersion
	err = gdb.Where("id = ?", versionID).First(&lv).Error
	if err != nil {
		return nil, err
	}
	return &lv, nil
}

func (s *dPackage) GetVersionByPurl(purl string) (out *model.PackageVersion, err error) {
	var lv model.PackageVersion
	err = gdb.Where("purl = ?", purl).First(&lv).Error
	if err != nil {
		return nil, err
	}
	return &lv, nil
}
