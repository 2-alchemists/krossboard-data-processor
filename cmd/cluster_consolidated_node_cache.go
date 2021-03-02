package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"time"
)


type NodeUsageDb struct {
	Path string
	Data *cache.Cache
}


func NewNodeUsageDB(path string, createIfNotExist bool) (*NodeUsageDb, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err)  && createIfNotExist {
		file, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
	}

	return &NodeUsageDb{
		Path: path,
		Data: cache.New(365 * 24 * time.Hour, 10*time.Minute),
	}, nil
}

func (m *NodeUsageDb) Load() error {
	itemsRaw, err := ioutil.ReadFile(m.Path)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed reading file %v", m.Path))
	}

	if len(itemsRaw) == 0 {
		return nil
	}

	items := &map[string]cache.Item{}
	err = json.Unmarshal(itemsRaw, items)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed decoding cache data %v", string(itemsRaw)))
	}

	m.Data = cache.NewFrom(365 * 24 * time.Hour, 10*time.Minute, *items)
	return nil
}


func (m *NodeUsageDb) Save() error {
	itemsRaw, err := json.Marshal(m.Data.Items())
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed encoding item data %v", m.Data.Items()))
	}

	err = ioutil.WriteFile(m.Path, itemsRaw, 0644)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed wrting file %v", m.Data.Items()))
	}

	return nil
}
