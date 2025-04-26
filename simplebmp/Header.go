
package simplebmp

import (
	"encoding/binary"
	"io"
	
)

type Header struct {
    
    Signature string // BMP Signature (BM)
    
    FileSize uint32 // Total file size
    
    Reserved uint32 // Reserved (0)
    
    DataOffset uint32 // Offset to image data
    
}

func (s *Header) Read(r io.Reader) error {
	var err error

    
	
	
	b := make([]byte, 2)
	
	   _, err = r.Read(b)
	if err != nil {
		return err
	}
	   s.Signature = string(b)
	

    
	
	err = binary.Read(r, binary.LittleEndian, &s.FileSize)
	if err != nil {
		return err
	}
	

    
	
	err = binary.Read(r, binary.LittleEndian, &s.Reserved)
	if err != nil {
		return err
	}
	

    
	
	err = binary.Read(r, binary.LittleEndian, &s.DataOffset)
	if err != nil {
		return err
	}
	

    

    return nil
}

func (s *Header) Write(w io.Writer) error {
	var err error

     
	 
	  _,err = w.Write([]byte(s.Signature))
	  if err!= nil{
		return err
	  }
	  
      
	 
	  err = binary.Write(w, binary.LittleEndian, s.FileSize)
	  if err!= nil{
		return err
	  }
	  
      
	 
	  err = binary.Write(w, binary.LittleEndian, s.Reserved)
	  if err!= nil{
		return err
	  }
	  
      
	 
	  err = binary.Write(w, binary.LittleEndian, s.DataOffset)
	  if err!= nil{
		return err
	  }
	  
      

	return nil
}

