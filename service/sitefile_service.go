package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"s2qt/util"
)

type SiteFileService struct {
	Paths *util.AppPaths
}

func NewSiteFileService() (*SiteFileService, error) {
	paths, err := util.GetAppPaths()
	if err != nil {
		return nil, err
	}

	return &SiteFileService{
		Paths: paths,
	}, nil
}

func (s *SiteFileService) PrepareSiteLogoFile(srcPath string) (string, error) {
	srcPath = strings.TrimSpace(srcPath)
	if srcPath == "" {
		return "", fmt.Errorf("로고 원본 경로가 비어 있습니다")
	}

	if s.Paths == nil {
		return "", fmt.Errorf("app paths가 비어 있습니다")
	}

	targetPath := strings.TrimSpace(s.Paths.SiteLogoFile)
	if targetPath == "" {
		targetPath = filepath.Join("var", "image", "site_logo.png")
	}

	srcAbs, _ := filepath.Abs(srcPath)
	dstAbs, _ := filepath.Abs(targetPath)

	if srcAbs == dstAbs {
		return targetPath, nil
	}

	return CopyFileToFixedPath(srcPath, targetPath)
}
