package dao

import (
	"github.com/alayou/techstack/model"
	"gorm.io/gorm"
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

// CreateOrUpdatePackage 创建或更新包
// 如果包已存在（根据 purl_type 和 name 判断），则更新 updated_at 字段
// 如果包不存在，则创建新包
func (s *dPackage) CreateOrUpdatePackage(pkg *model.Package) error {
	var existingPkg model.Package
	err := gdb.Where("purl_type = ? AND name = ?", pkg.PurlType, pkg.Name).First(&existingPkg).Error

	if err == nil {
		// 包已存在，更新 updated_at
		return gdb.Model(&existingPkg).Update("updated_at", pkg.UpdatedAt).Error
	} else if err == gorm.ErrRecordNotFound {
		// 包不存在，创建新包
		return gdb.Create(pkg).Error
	}
	return err
}

// BatchCreateOrUpdatePackages 批量创建或更新包
// 返回成功和失败的数量
func (s *dPackage) BatchCreateOrUpdatePackages(pkgs []*model.Package) (successCount, failedCount int) {
	for _, pkg := range pkgs {
		if err := s.CreateOrUpdatePackage(pkg); err != nil {
			failedCount++
		} else {
			successCount++
		}
	}
	return
}

// GetPackagesByPurlType 根据 purl_type 获取包列表
func (s *dPackage) GetPackagesByPurlType(purlType string, offset, limit int) (ls []*model.Package, total int64, err error) {
	var count int64
	err = gdb.Model(&model.Package{}).Where("purl_type = ?", purlType).Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	err = gdb.Where("purl_type = ?", purlType).Offset(offset).Limit(limit).Find(&ls).Error
	if err != nil {
		return nil, 0, err
	}
	return ls, count, nil
}
