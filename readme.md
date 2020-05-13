# i2r
I2R is library which handles accessing IMAP server over HTTP using websockets.
It also supports sending messages over SMTP.

It has been created because [imapapi](https://github.com/andris9/imapapi) is licensed on AGPL which is unacceptable in my project.
It's also written in JS, which is bad for embbeding IMO(unless you use JS as main language with something like cordova or nativescript). Single binary produced by go should do better in cases like android application.