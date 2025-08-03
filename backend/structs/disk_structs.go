package structs

import (
    "encoding/binary"
    "os"
)

type MBR struct {
    Mbr_tamano int32
    Mbr_fecha_creacion [16]byte
    Mbr_dsk_signature int32
    Dsk_fit byte
    Mbr_partitions [4]Partition
}

type Partition struct {
    Part_status byte
    Part_type byte
    Part_fit byte
    Part_start int32
    Part_s int32
    Part_name [16]byte
    Part_correlative int32
    Part_id [4]byte
}

type EBR struct {
    Part_mount byte
    Part_fit byte
    Part_start int32
    Part_s int32
    Part_next int32
    Part_name [16]byte
}

func (m *MBR) WriteToFile(file *os.File) error {
    file.Seek(0, 0)
    return binary.Write(file, binary.LittleEndian, m)
}

func LeerMBR(archivo *os.File) (MBR, error) {
    archivo.Seek(0, 0)
    var mbr MBR
    err := binary.Read(archivo, binary.LittleEndian, &mbr)
    return mbr, err
}