package xml

import (
	"encoding/xml"
	"jtso/logger"
)

type RawData struct {
	RtrName    string
	Family     string
	IfInfo     *Ifdesc
	HwInfo     *Hw
	LacpInfo   *Lacp
	LacpDigest *LacpDigest
}

// Struct for unmarshalling version
type Version struct {
	XMLName xml.Name `xml:"software-information"`
	Model   string   `xml:"product-model"`
	Ver     string   `xml:"junos-version"`
}

// Structs for unmarshalling interfaces descriptions
type Ifdesc struct {
	XMLName   xml.Name `xml:"interface-information"`
	Physicals []Phy    `xml:"physical-interface"`
	Logicals  []Log    `xml:"logical-interface"`
}

type Phy struct {
	XMLName xml.Name `xml:"physical-interface"`
	Name    string   `xml:"name"`
	Desc    string   `xml:"description"`
}

type Log struct {
	XMLName xml.Name `xml:"logical-interface"`
	Name    string   `xml:"name"`
	Desc    string   `xml:"description"`
}

// structs for umarshalling chassis hw
type Hw struct {
	XMLName xml.Name `xml:"chassis-inventory"`
	Chassis Chassis  `xml:"chassis"`
}

type Chassis struct {
	XMLName xml.Name `xml:"chassis"`
	Desc    string   `xml:"description"`
	Modules []Module `xml:"chassis-module"`
}

type Module struct {
	XMLName xml.Name `xml:"chassis-module"`
	Name    string   `xml:"name"`
	Desc    string   `xml:"description"`
	SubMods []SubMod `xml:"chassis-sub-module"`
}

type SubMod struct {
	XMLName    xml.Name    `xml:"chassis-sub-module"`
	Name       string      `xml:"name"`
	Desc       string      `xml:"description"`
	SubSubMods []SubSubMod `xml:"chassis-sub-sub-module"`
}

type SubSubMod struct {
	XMLName       xml.Name       `xml:"chassis-sub-sub-module"`
	Name          string         `xml:"name"`
	Desc          string         `xml:"description"`
	SubSubSubMods []SubSubSubMod `xml:"chassis-sub-sub-sub-module"`
}

type SubSubSubMod struct {
	XMLName xml.Name `xml:"chassis-sub-sub-sub-module"`
	Name    string   `xml:"name"`
	Desc    string   `xml:"description"`
}

type Lacp struct {
	XMLName xml.Name  `xml:"lacp-interface-information-list"`
	LacpInt []LacpInt `xml:"lacp-interface-information"`
}

type LacpInt struct {
	XMLName   xml.Name    `xml:"lacp-interface-information"`
	LacpHead  LacpHead    `xml:"lag-lacp-header"`
	LacpProto []LacpProto `xml:"lag-lacp-protocol"`
}

type LacpHead struct {
	XMLName xml.Name `xml:"lag-lacp-header"`
	LagName string   `xml:"aggregate-name"`
}

type LacpProto struct {
	XMLName xml.Name `xml:"lag-lacp-protocol"`
	Name    string   `xml:"name"`
}

// easy to parse LACP data
type LacpDigest struct {
	LacpMap map[string]string
}

// Parsing function for version
func ParseVersion(s string) (*Version, error) {
	logger.HandlePanic()
	var i Version
	// convert in byte array
	b := []byte(s)
	// unmarshall xml string
	err := xml.Unmarshal(b, &i)
	return &i, err
}

// Parsing function for interfaces description
func ParseIfdesc(s string) (*Ifdesc, error) {
	logger.HandlePanic()
	var i Ifdesc
	// convert in byte array
	b := []byte(s)
	// unmarshall xml string
	err := xml.Unmarshal(b, &i)
	return &i, err
}

// Parsing function for chassis hw
func ParseChassis(s string) (*Hw, error) {
	logger.HandlePanic()
	var i Hw
	// convert in byte array
	b := []byte(s)
	// unmarshall xml string
	err := xml.Unmarshal(b, &i)
	return &i, err
}

// Parsing function for Lacp interface
func ParseLacp(s string) (*Lacp, *LacpDigest, error) {
	logger.HandlePanic()
	var i Lacp
	data := new(LacpDigest)
	data.LacpMap = make(map[string]string)
	// convert in byte array
	b := []byte(s)
	// unmarshall xml string
	err := xml.Unmarshal(b, &i)
	// now parse reply
	if err == nil {
		for _, l := range i.LacpInt {
			for _, n := range l.LacpProto {
				data.LacpMap[n.Name] = l.LacpHead.LagName
			}
		}
	}

	return &i, data, err
}
