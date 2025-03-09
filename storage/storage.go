// storage is a key-value database and its interfaces indeed
// the information of block will be saved in storage

package storage

import (
	"blockEmulator/Block"
	"errors"
	"github.com/boltdb/bolt"
	"log"
	"os"
	"strconv"
)

type Storage struct {
	dbFilePath            string // path to the database
	blockBucket           string // bucket in bolt database
	HeaderBucket          string // bucket in bolt database
	newestBlockHashBucket string // bucket in bolt database
	DataBase              *bolt.DB
}

func NewStorage(port string, shard uint64) *Storage {
	_, errStat := os.Stat("./record")
	if os.IsNotExist(errStat) {
		errMkdir := os.Mkdir("./record", os.ModePerm)
		if errMkdir != nil {
			log.Panic(errMkdir)
		}
	} else if errStat != nil {
		log.Panic(errStat)
	}
	s := &Storage{
		dbFilePath:            "./record/" + port + "_" + strconv.FormatUint(shard, 10) + "_database",
		blockBucket:           "block",
		HeaderBucket:          "head",
		newestBlockHashBucket: "newestBlockHash",
	}
	// 检查数据库文件是否存在
	if _, err := os.Stat(s.dbFilePath); err == nil {
		// 文件存在，尝试删除它
		if err := os.Remove(s.dbFilePath); err != nil {
			log.Fatalf("Failed to remove existing database file: %s", err)
		} else {
			log.Println("Old database removed")
		}
	}
	db, err := bolt.Open(s.dbFilePath, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	// create buckets
	_ = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(s.blockBucket))
		if err != nil {
			log.Panic("create blocksBucket failed")
		}

		_, err = tx.CreateBucketIfNotExists([]byte(s.HeaderBucket))
		if err != nil {
			log.Panic("create HeaderBucket failed")
		}

		_, err = tx.CreateBucketIfNotExists([]byte(s.newestBlockHashBucket))
		if err != nil {
			log.Panic("create newestBlockHashBucket failed")
		}

		return nil
	})
	s.DataBase = db
	return s

}

// update the newest block in the database
func (s *Storage) UpdateNewestBlock(newestBHash []byte) {
	err := s.DataBase.Update(func(tx *bolt.Tx) error {
		nbhBucket := tx.Bucket([]byte(s.newestBlockHashBucket))
		// the bucket has the only key "OnlyNewestBlock"
		err := nbhBucket.Put([]byte("OnlyNewestBlock"), newestBHash)
		if err != nil {
			log.Panic()
		}
		return nil
	})
	if err != nil {
		log.Panic()
	}
	//fmt.Println("The newest block is updated")
}

// add a head into the database
func (s *Storage) AddHeader(blockHash []byte, byteHead []byte) {
	err := s.DataBase.Update(func(tx *bolt.Tx) error {
		bHeadBucket := tx.Bucket([]byte(s.HeaderBucket))
		err := bHeadBucket.Put(blockHash, byteHead)
		if err != nil {
			log.Panic()
		}
		return nil
	})
	if err != nil {
		log.Panic()
	}
}

// add a block into the database
func (s *Storage) AddBlock(block Block.Block) {
	bHash := block.Hash().Bytes()
	byteHead := block.Head().EncodeH()
	byteBlock := block.Encode()
	err := s.DataBase.Update(func(tx *bolt.Tx) error {
		bbucket := tx.Bucket([]byte(s.blockBucket))
		err := bbucket.Put(bHash, byteBlock)
		if err != nil {
			log.Panic()
		}
		return nil
	})
	if err != nil {
		log.Panic()
	}
	s.AddHeader(bHash, byteHead)
	s.UpdateNewestBlock(bHash)
	//fmt.Println("Block is added")
}

// read a block from the database
func (s *Storage) GetBlock(bHash []byte) ([]byte, error) {
	var res []byte = nil
	err := s.DataBase.View(func(tx *bolt.Tx) error {
		bBucket := tx.Bucket([]byte(s.blockBucket))
		bEncoded := bBucket.Get(bHash)
		if bEncoded == nil {
			return errors.New("the block is not existed")
		}
		res = bEncoded
		return nil
	})
	return res, err
}

// read the Newest block hash
func (s *Storage) GetNewestBlockHash() ([]byte, error) {
	var nhb []byte
	err := s.DataBase.View(func(tx *bolt.Tx) error {
		bhBucket := tx.Bucket([]byte(s.newestBlockHashBucket))
		// the bucket has the only key "OnlyNewestBlock"
		nhb = bhBucket.Get([]byte("OnlyNewestBlock"))
		if nhb == nil {
			return errors.New("cannot find the newest block hash")
		}
		return nil
	})
	return nhb, err
}
