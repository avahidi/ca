package main

import (
	"os"
	"path"
	"time"
)

// Locator represents something the can be placed in the cache and later retrieved
type Locator interface {
	Location() (string, string)
}

// Cache represents the folder that is our cache system
type Cache struct {
	base string
}

func NewCache(base string) *Cache {
	return &Cache{base: base}
}

func (c Cache) file(q Locator) string {
	group, name := q.Location()
	return path.Join(c.base, group, name)
}

func (c Cache) folder(q Locator) string {
	group, _ := q.Location()
	return path.Join(c.base, group)
}

func (c Cache) Check(q Locator, maxAge int) (exists bool, isRecent bool) {
	info, err := os.Stat(c.file(q))
	if err != nil {
		return false, false
	}

	age := time.Since(info.ModTime()).Minutes()
	return true, age <= float64(maxAge)
}

func (c Cache) Read(q Locator) ([]byte, error) {
	return os.ReadFile(c.file(q))
}

func (c Cache) Write(q Locator, content []byte) error {
	folder := c.folder(q)
	if err := os.MkdirAll(folder, 0700); err != nil {
		return err
	}

	filename := c.file(q)
	return os.WriteFile(filename, content, 0600)
}
