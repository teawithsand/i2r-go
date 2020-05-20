package i2r

import (
	"github.com/teawithsand/rgz"
)

// TextError is error made of text, which can be 
type TextError struct {
	Text string `json:"text"`
}

func (e *TextError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Text
}

// TextErrorMsgMapper mapps all errors into text errors.
// It should be mounted on any marshaler before using it.
// It's resposbile for handling errors returned from IMAP handler.
func TextErrorMsgMapper() rgz.MsgMapper {
	return rgz.MsgMapper(func(msg interface{}) (newMsg interface{}, err error){
		if err, ok := msg.(error); ok{
			newMsg = TextError{
				Text: err.Error(),
			}
		} else {
			newMsg = msg
		}
		return
	})
}