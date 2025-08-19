package structs

import (
    "encoding/binary"
    "os"
)

var Particiones_Montadas []PartitionMount

type PartitionMount struct {
    Id string
    Path string
    Partition Partition
    UserFile string
}

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

type SuperBloque struct {
    S_filesystem_type int32
    S_inodes_count int32
    S_blocks_count int32
    S_free_blocks_count int32
    S_free_inodes_count int32
    S_mtime [16]byte
    S_umtime [16]byte
    S_mnt_count int32
    S_magic int32
    S_inode_s int32
    S_block_s int32
    S_first_ino int32
    S_first_blo int32
    S_bm_inode_start int32
    S_bm_block_start int32
    S_inode_start int32
    S_block_start int32
}

type Inodo struct {
    I_uid int32
    I_gid int32
    I_s int32
	I_atime [17]byte
	I_ctime [17]byte
	I_mtime [17]byte
	I_block [15]int32
	I_type  [1]byte
	I_perm  [3]byte
}

type BContent struct {
    B_name [12]byte
    B_inodo int32
}

type BCarpeta struct {
    B_content [4]BContent
}

type BArchivo struct {
    B_content [64]byte
}

type BApuntadores struct {
    B_pointers [16]int32
}

type Bitmap []byte

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

func GetMountedPartitionByID(id string, mbr MBR) Partition {
    for _, pm := range Particiones_Montadas {
        if pm.Id == id {
            return pm.Partition
        }
    }
    return Partition{}
}

func GetDiskPathByID(id string) string {
    for _, pm := range Particiones_Montadas {
        if pm.Id == id {
            return pm.Path
        }
    }
    return ""
}