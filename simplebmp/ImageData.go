
package simplebmp

import (
	"encoding/binary"
	"io"
	
)

type ImageData struct {
    
    Width uint32 // Image width
    
    Height uint32 // Image height
    
    PixelData []byte // RGB pixel data
    
}

func (s *ImageData) Read(r io.Reader) error {
	var err error

    
	
	err = binary.Read(r, binary.LittleEndian, &s.Width)
	if err != nil {
		return err
	}
	

    
	
	err = binary.Read(r, binary.LittleEndian, &s.Height)
	if err != nil {
		return err
	}
	

    
	

    

    return nil
}

func (s *ImageData) Write(w io.Writer) error {
	var err error

     
	 
	  err = binary.Write(w, binary.LittleEndian, s.Width)
	  if err!= nil{
		return err
	  }
	  
      
	 
	  err = binary.Write(w, binary.LittleEndian, s.Height)
	  if err!= nil{
		return err
	  }
	  
      
	 
      

	return nil
}

