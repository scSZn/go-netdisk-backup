package scanner

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"backup/internal/dao"
	"backup/pkg/database"
	"backup/pkg/logger"
	"backup/pkg/util"
)

var Manager = new(scannerManager)

type scannerManager struct {
	lock     sync.Mutex
	scanners []*Scanner
}

func (s *scannerManager) Start(ctx context.Context) {
	backupPaths := dao.NewBackupPathDao(ctx, database.DB).GetAll()

	for _, path := range backupPaths {
		scanner, err := NewScanner(util.NewContext(), path.AbsPath)
		if err != nil {
			logger.Logger.WithField("pathInfo", path).WithError(err).Error("add scanner fail")
			continue
		}
		s.scanners = append(s.scanners, scanner)
	}

	go func() {
		// TODO: 做成可调整的
		ticker := time.NewTicker(300 * time.Second)
		for {
			select {
			case <-ticker.C:
				s.lock.Lock()
				for _, scanner := range s.scanners {
					go scanner.ScanAndUpload()
				}
				s.lock.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *scannerManager) Add(scanner *Scanner) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.scanners = append(s.scanners, scanner)
}

func (s *scannerManager) Remove(root string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var index = -1
	for i, scanner := range s.scanners {
		if filepath.Clean(scanner.root) == filepath.Clean(root) {
			index = i
			scanner.Cancel() // 取消扫描
			break
		}
	}

	if index != -1 {
		s.scanners = append(s.scanners[:index], s.scanners[index+1:]...)
	}
}
