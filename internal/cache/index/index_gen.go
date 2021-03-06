/**
* Copyright 2018 Comcast Cable Communications Management, LLC
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
* http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package index

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Index) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "CacheSize":
			z.CacheSize, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "ObjectCount":
			z.ObjectCount, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Objects":
			var zb0002 uint32
			zb0002, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Objects == nil {
				z.Objects = make(map[string]*Object, zb0002)
			} else if len(z.Objects) > 0 {
				for key := range z.Objects {
					delete(z.Objects, key)
				}
			}
			for zb0002 > 0 {
				zb0002--
				var za0001 string
				var za0002 *Object
				za0001, err = dc.ReadString()
				if err != nil {
					return
				}
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					za0002 = nil
				} else {
					if za0002 == nil {
						za0002 = new(Object)
					}
					err = za0002.DecodeMsg(dc)
					if err != nil {
						return
					}
				}
				z.Objects[za0001] = za0002
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Index) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "CacheSize"
	err = en.Append(0x83, 0xa9, 0x43, 0x61, 0x63, 0x68, 0x65, 0x53, 0x69, 0x7a, 0x65)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.CacheSize)
	if err != nil {
		return
	}
	// write "ObjectCount"
	err = en.Append(0xab, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.ObjectCount)
	if err != nil {
		return
	}
	// write "Objects"
	err = en.Append(0xa7, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Objects)))
	if err != nil {
		return
	}
	for za0001, za0002 := range z.Objects {
		err = en.WriteString(za0001)
		if err != nil {
			return
		}
		if za0002 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = za0002.EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Index) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "CacheSize"
	o = append(o, 0x83, 0xa9, 0x43, 0x61, 0x63, 0x68, 0x65, 0x53, 0x69, 0x7a, 0x65)
	o = msgp.AppendInt64(o, z.CacheSize)
	// string "ObjectCount"
	o = append(o, 0xab, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	o = msgp.AppendInt64(o, z.ObjectCount)
	// string "Objects"
	o = append(o, 0xa7, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73)
	o = msgp.AppendMapHeader(o, uint32(len(z.Objects)))
	for za0001, za0002 := range z.Objects {
		o = msgp.AppendString(o, za0001)
		if za0002 == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = za0002.MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Index) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "CacheSize":
			z.CacheSize, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "ObjectCount":
			z.ObjectCount, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Objects":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Objects == nil {
				z.Objects = make(map[string]*Object, zb0002)
			} else if len(z.Objects) > 0 {
				for key := range z.Objects {
					delete(z.Objects, key)
				}
			}
			for zb0002 > 0 {
				var za0001 string
				var za0002 *Object
				zb0002--
				za0001, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					za0002 = nil
				} else {
					if za0002 == nil {
						za0002 = new(Object)
					}
					bts, err = za0002.UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
				z.Objects[za0001] = za0002
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Index) Msgsize() (s int) {
	s = 1 + 10 + msgp.Int64Size + 12 + msgp.Int64Size + 8 + msgp.MapHeaderSize
	if z.Objects != nil {
		for za0001, za0002 := range z.Objects {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001)
			if za0002 == nil {
				s += msgp.NilSize
			} else {
				s += za0002.Msgsize()
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Object) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "key":
			z.Key, err = dc.ReadString()
			if err != nil {
				return
			}
		case "expiration":
			z.Expiration, err = dc.ReadTime()
			if err != nil {
				return
			}
		case "lastwrite":
			z.LastWrite, err = dc.ReadTime()
			if err != nil {
				return
			}
		case "lastaccess":
			z.LastAccess, err = dc.ReadTime()
			if err != nil {
				return
			}
		case "size":
			z.Size, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "value":
			z.Value, err = dc.ReadBytes(z.Value)
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Object) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "key"
	err = en.Append(0x86, 0xa3, 0x6b, 0x65, 0x79)
	if err != nil {
		return
	}
	err = en.WriteString(z.Key)
	if err != nil {
		return
	}
	// write "expiration"
	err = en.Append(0xaa, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return
	}
	err = en.WriteTime(z.Expiration)
	if err != nil {
		return
	}
	// write "lastwrite"
	err = en.Append(0xa9, 0x6c, 0x61, 0x73, 0x74, 0x77, 0x72, 0x69, 0x74, 0x65)
	if err != nil {
		return
	}
	err = en.WriteTime(z.LastWrite)
	if err != nil {
		return
	}
	// write "lastaccess"
	err = en.Append(0xaa, 0x6c, 0x61, 0x73, 0x74, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73)
	if err != nil {
		return
	}
	err = en.WriteTime(z.LastAccess)
	if err != nil {
		return
	}
	// write "size"
	err = en.Append(0xa4, 0x73, 0x69, 0x7a, 0x65)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.Size)
	if err != nil {
		return
	}
	// write "value"
	err = en.Append(0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Value)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Object) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "key"
	o = append(o, 0x86, 0xa3, 0x6b, 0x65, 0x79)
	o = msgp.AppendString(o, z.Key)
	// string "expiration"
	o = append(o, 0xaa, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendTime(o, z.Expiration)
	// string "lastwrite"
	o = append(o, 0xa9, 0x6c, 0x61, 0x73, 0x74, 0x77, 0x72, 0x69, 0x74, 0x65)
	o = msgp.AppendTime(o, z.LastWrite)
	// string "lastaccess"
	o = append(o, 0xaa, 0x6c, 0x61, 0x73, 0x74, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73)
	o = msgp.AppendTime(o, z.LastAccess)
	// string "size"
	o = append(o, 0xa4, 0x73, 0x69, 0x7a, 0x65)
	o = msgp.AppendInt64(o, z.Size)
	// string "value"
	o = append(o, 0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendBytes(o, z.Value)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Object) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "key":
			z.Key, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "expiration":
			z.Expiration, bts, err = msgp.ReadTimeBytes(bts)
			if err != nil {
				return
			}
		case "lastwrite":
			z.LastWrite, bts, err = msgp.ReadTimeBytes(bts)
			if err != nil {
				return
			}
		case "lastaccess":
			z.LastAccess, bts, err = msgp.ReadTimeBytes(bts)
			if err != nil {
				return
			}
		case "size":
			z.Size, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "value":
			z.Value, bts, err = msgp.ReadBytesBytes(bts, z.Value)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Object) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.Key) + 11 + msgp.TimeSize + 10 + msgp.TimeSize + 11 + msgp.TimeSize + 5 + msgp.Int64Size + 6 + msgp.BytesPrefixSize + len(z.Value)
	return
}
