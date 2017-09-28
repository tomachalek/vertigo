# vertigo

The program is intended for parsing so called *corpus vertical files*, which are basically SGML files where structural information is realized by custom tags (each tag on its own line) and token information (again, each token on its own line) is realized via tab-separated values (e.g. *word*[tab]*lemma*[tab]*tag*). The file looks like this:

```
<doc id="adams-restaurant_at_the" lang="en" version="00" wordcount="54066">
<div author="Adams, Douglas" title="The Restaurant at the End of the Universe" group="Core" publisher="" pubplace="" pubyear="1980" pubmonth="" origyear="" isbn="" txtype="fiction" comment="" original="Yes" srclang="en" translator="" transsex="" authsex="M" lang_var="en-GB" id="en:adams-restaurant_na_ko:0" wordcount="54066">
<p id="en:adams-restaurant_na_ko:0:1">
<s id="en:adams-restaurant_na_ko:0:1:1">
The     the     DT
Restaurant      Restaurant      NP
at      at      IN
the     the     DT
End     end     NN
of      of      IN
the     the     DT
Universe        universe        NN
</s>
</p>
<p id="en:adams-restaurant_na_ko:0:2">
<s id="en:adams-restaurant_na_ko:0:2:1">
There   there   EX
is      be      VBZ
a       a       DT
theory  theory  NN
...
```

Vertigo parses the files using two goroutines. First one (aka the "producer") parses file line by line and fills in a channel, the second one (aka the "consumer") passes parsed lines to a *LineProcessor* implementation obtained from an external caller.

The *LineProcessor* interface is the following:

```go
type LineProcessor interface {
	ProcToken(token *Token)
	ProcStruct(strc *Structure)
	ProcStructClose(strc *StructureClose)
}
```
