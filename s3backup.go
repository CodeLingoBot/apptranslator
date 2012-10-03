// This code is under BSD license. See license-bsd.txt
package main

// TODO: delete old backup files and only keep the last, say, 64?

import (
	_ "fmt"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"log"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var backupFreq = 4 * time.Hour
var bucketDelim = "/"

type BackupConfig struct {
	AwsAccess string
	AwsSecret string
	Bucket    string
	S3Dir     string
	LocalDir  string
}

// removes "/" if exists and adds delim if missing
func sanitizeDirForList(dir, delim string) string {
	if strings.HasPrefix(dir, "/") {
		dir = dir[1:]
	}
	if !strings.HasSuffix(dir, delim) {
		dir = dir + delim
	}
	return dir
}

func listBackupFiles(config *BackupConfig, max int) (*s3.ListResp, error) {
	auth := aws.Auth{config.AwsAccess, config.AwsSecret}
	b := s3.New(auth, aws.USEast).Bucket(config.Bucket)
	dir := sanitizeDirForList(config.S3Dir, bucketDelim)
	return b.List(dir, bucketDelim, "", max)
}

func s3Put(config *BackupConfig, local, remote string, public bool) error {
	localf, err := os.Open(local)
	if err != nil {
		return err
	}
	defer localf.Close()
	localfi, err := localf.Stat()
	if err != nil {
		return err
	}

	auth := aws.Auth{config.AwsAccess, config.AwsSecret}
	b := s3.New(auth, aws.USEast).Bucket(config.Bucket)

	acl := s3.Private
	if public {
		acl = s3.PublicRead
	}

	contType := mime.TypeByExtension(path.Ext(local))
	if contType == "" {
		contType = "binary/octet-stream"
	}

	err = b.PutBucket(acl)
	if err != nil {
		return err
	}
	return b.PutReader(remote, localf, localfi.Size(), contType, acl)
}

// tests if s3 credentials are valid and aborts if aren't
func ensureValidConfig(config *BackupConfig) {
	if !PathExists(config.LocalDir) {
		log.Fatalf("Invalid s3 backup: directory to backup '%s' doesn't exist\n", config.LocalDir)
	}

	if !strings.HasSuffix(config.S3Dir, bucketDelim) {
		config.S3Dir += bucketDelim
	}
	_, err := listBackupFiles(config, 10)
	if err != nil {
		log.Fatalf("Invalid s3 backup: bucket.List failed %s\n", err.Error())
	}
}

// Return true if a backup file with given sha1 content has already been uploaded
// Grabs 10 newest files and checks if sha1 is part of the name, on the theory
// that if the content hasn't changed, the last backup file should have
// the same content, so we don't need to check all files
func alreadyUploaded(config *BackupConfig, sha1 string) bool {
	rsp, err := listBackupFiles(config, 10)
	if err != nil {
		logger.Errorf("alreadyUploaded(): listBackupFiles() failed with %s", err.Error())
		return false
	}
	for _, key := range rsp.Contents {
		if strings.Contains(key.Key, sha1) {
			//fmt.Printf("Backup file with sha1 %s already exists: %s\n", sha1, key.Key)
			return true
		}
	}
	//fmt.Printf("Backup file with sha1 %s hasn't been uploaded yet\n")
	return false
}

func doBackup(config *BackupConfig) {
	zipLocalPath := filepath.Join(os.TempDir(), "apptranslator-tmp-backup.zip")
	//fmt.Printf("zip file name: %s\n", zipLocalPath)
	// TODO: do I need os.Remove() won't os.Create() over-write the file anyway?
	os.Remove(zipLocalPath) // remove before trying to create a new one, just in cased
	err := CreateZipWithDirContent(zipLocalPath, config.LocalDir)
	defer os.Remove(zipLocalPath)
	if err != nil {
		return
	}
	sha1, err := FileSha1(zipLocalPath)
	if err != nil {
		return
	}
	if alreadyUploaded(config, sha1) {
		logger.Noticef("s3 backup not done because data hasn't changed")
		return
	}
	timeStr := time.Now().Format("060102_1504_")
	zipS3Path := path.Join(config.S3Dir, timeStr+sha1+".zip")

	if err = s3Put(config, zipLocalPath, zipS3Path, true); err != nil {
		logger.Errorf("s3Put of '%s' to '%s' failed with %s", zipLocalPath, zipS3Path, err.Error())
		return
	}

	logger.Noticef("s3 backup of '%s' to '%s'", zipLocalPath, zipS3Path)
}

func BackupLoop(config *BackupConfig) {
	ensureValidConfig(config)
	for {
		doBackup(config)
		time.Sleep(backupFreq)
	}
}
