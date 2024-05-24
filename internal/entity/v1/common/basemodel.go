package common

import "time"

type BaseModel struct {
	Id        int       `json:"id" storm:"id,increment,index,unique"`
	CreatedAt time.Time `json:"createdAt" storm:"index"`
	UpdatedAt time.Time `json:"updatedAt" storm:"index"`
}
