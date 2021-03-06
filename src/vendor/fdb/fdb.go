package fdb

import (
	"packet"
	"log"
	"sync"
	"time"
	"container/list"
	"bytes"
)
type I interface{

}
type FdbMacNode struct {
	client *Client
	mac packet.MAC
	ft uint64
}
func NewFdbMacNode(m packet.MAC, c *Client) *FdbMacNode {
	return &FdbMacNode{
		client : c,
		mac : m,
		ft : FdbTick,
	}
}
func (fmt *FdbMacNode) GetClient() *Client{
	return fmt.client
}

type FDB struct{
	lock *sync.RWMutex
	mactable map[packet.MAC]I
}
var myfdb *FDB
var FdbTick uint64
func Fdb() *FDB {
	return myfdb
}
func FdbMacTable() map[packet.MAC]I {
	return Fdb().mactable
}
func init() {
	myfdb = &FDB{
		lock : new(sync.RWMutex),
		mactable : make(map[packet.MAC]I),
	}
	InitClientList()
	go fdbtick()
}
func fdbtick() {
	FdbTick = 0
	tt := time.Tick(time.Second * 3)
	for _ = range tt {
		FdbTick = FdbTick + 1
	}
}
func (f *FDB) Get(m packet.MAC) (I, bool) {
	defer f.lock.RUnlock()
	f.lock.RLock()
	c, ok := f.mactable[m]
	return c, ok
}
func (f *FDB) Add(m packet.MAC, c *Client) {
	fmn := NewFdbMacNode(m, c)	
	f.lock.Lock()
	f.mactable[m] = fmn
	f.lock.Unlock()
}
func (f *FDB) Del(m packet.MAC) {
	f.lock.Lock()
	delete(f.mactable, m)
	f.lock.Unlock()
}
func (f *FDB) DelFmnByClient(c *Client) {	
	for m, i := range FdbMacTable(){
		fmn := i.(*FdbMacNode)
		if fmn.client == c {
			Fdb().Del(m)
		}
	}
}
func MtShowAll(){
	for m, i := range FdbMacTable() {
		fmn := i.(*FdbMacNode)
		log.Printf("mac =%s, c =%s\n", m.String(), fmn.client.LocalString())
	}
}
func ShowClientMac() map[string][]string {
	mt := make(map[string][]string)
	var n *list.Element
	for e := GetClientList().Front(); e != nil ; e = n{
		n = e.Next()
		c, ok := e.Value.(*Client)
		if !ok {
			log.Printf("========== can't happend======\n")
			GetClientList().Remove(e)
			continue			
		}

		for m, i := range FdbMacTable(){
			if fmn, ok := i.(*FdbMacNode); ok {
				if fmn.client == c {
					caddr := fmn.client.LocalString()
					//log.Printf("========== caddr key=%s, len=%d======\n", caddr, len(caddr))
					cb :=[]byte(caddr)
					index := bytes.IndexByte(cb, 0)
					//log.Printf("index =%d\n",index)
					if index > 0 && index < len(caddr) {
						caddr = string(cb[0:index])
					 	log.Println(caddr)
					}
					mt[caddr] = append(mt[caddr], m.String())
				}
			}else{
				log.Printf("========== can't happend, mac=%s relate to unormal fmn======\n", m.String())
				//TODO : del m from fdb mactable
				Fdb().Del(m)
			}
		}

	}
	//log.Println(mt)
	return mt
}