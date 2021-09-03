---
layout: post
title: Kubernetes requests, limits and throttling (cesky)
comments: true
---

## osnova
- [requesty, limity: lehke vysvetleni](#cast-1-requesty-a-limity)
- [zakladni doporuceni jak je requesty / limity nastavit](#zakladni-doporuceni)
- [throttling v kube: co to je a jake ma dusledky](#cast-2-cpu-throttling)
- [aplikace citlive na response time: jak to nastavit o nich](#cast-3-aplikace-citlive-na-response-time)
- [extras: prometheus queries](#extras-prometheus-queries) 
- zdroje informaci


# Cast #1: requesty a limity
V kostce slouzi requesty a limity k tomu, aby bylo v kube clusteru mozne nastavit nejake "rozumne" souziti vice aplikaci. A to tak aby za beznych okolnosti jedna aplikace nemohla "utlacovat" ostatni, scheduler dokazal pody aplikaci rozmistit na worker nody, dulezitym aplikacim bylo mozne zajistit dostatecny vykon a stabilitu, eventuelne prioritu. Dale rozumne nastaveni requestu a limitu umozni prijatelnou utilizaci zdroju na serverech (densitu).  

Z predchoziho popisu je naprosto zjevne, ze neni mozne splnit vsechno najednou, takze cely problem je o nalezeni dobreho kompromisu vhodneho pro nas use cases.

## typy zdroju
CPU je takzvane "compressible resource", coz znamena ze neni-li dostatek cpu tak pody dale bezi, jen je cpu prideleno podle nejakych pravidel (ktere vychazeji z qos tridy podu a jeho CPU requestu) a nektere pody pobezi pomalejil, protoze se na ne jiz nedostane dostatecny dil cpu.

Memory je "uncompressible resource" a v pripade, ze by na worker nodu chtely containery spotrebovat vice pameti nez je k dispozici prijde dohry OOM Killer, ktery podle definovanych pravidel situaci vyresi zabitim nekterych podu. Pokud container v podu prekroci svuj vlastni memory limit bude pod take zabit.

## zakladni doporuceni
Uplne nejzakladnejsi rada jak nastavit requesty/limity by mohla znit zhruba takto: "Nastavte request na cca 80% toho co vyzaduje bezny provoz aplikace a limit tak aby pokryl beznou spicku". Limit pro pamet bude spis vychazet ze skutecnych naroku aplikace na pamet, ktere obvykle muzete bud nastavit nebo odhadnout. Co se tyce pameti, tak limit musi byt nataven tak aby se tam "vesla", jinak bude dochazet k opakovanym restartum podu kdyz vyuzita pamet prekroci limit.

Konkretni hodnoty nenastavujme jen tak "od oka", ale napriklad nasledujicim postupem
1. pustim pod s rozumne odhadnutym requestm (cpu i memory) a bez limitu
2. ve spolupraci s nejakm nastrojem na stress testing postupne vytezuji aplikaci a sbiram metriky
    - container_memory_working_set_bytes
    - container_cpu_usage_seconds_total
    - container_cpu_cfs_throttled_seconds_total  


    rate techto metrik nam pomuze urcit vhodny limit (ukazkove prometheus queries jsou v [zaveru clanku](#extras-prometheus-queries))
3. limit pro pamet musi byt > `rate(container_memory_working_set_bytes)`, jinak bude pod zabit
4. limit pro cpu muze byt o neco nizsi nez nejvyssi namereny `rate(container_cpu_usage_seconds_total)`, jak moc niz zalezi asi zejmena na citlivosti aplikace na dobu odezvy. Je-li rychlost a konzistence odezvy dulezita, tak bychom meli cilit na nizky throttling (`rate(ontainer_cpu_cfs_throttled_seconds_total)`), jak presne nizky je asi trochu filozoficka otazka a zalezi asi na prioritach (performance vs density). Viz: https://youtu.be/UE7QX98-kO0?t=2249)


## hranicni pripady a dusledky
1. **prilis velke requesty:**  
Muze dojit k situaci, ze pody nebudou spusteny, protoze z pohledu scheduleru se jiz na zadbny worker node nevejdou (nezavisle na skutecnem vyuziti cpu/pameti na nodu).  
Dalsi problem, ktery prilis vysoke requesty zpusobi je nizke vyuziti compute resources (proste se na jednotlive worker nody vejde mene podu a pokud skutecne vyuziti bude nizsi nez requesty tak vetsi ci mensi cast cpu/memory resources bude lezet ladem)
2. **prilis nizky memory limit:**  
 aplikace vycerpa svoji pridelenou pamet a pote bude zabita. Pokud je mozne pametove naroky aplikace nastavit nebo odhadnout, tak je vhodne adekvatne k tomu nastavit limity containeru.
3. **prilis nizky cpu limit:**  
Dojde k throttlingu. To znamena, ze container vycerpa svuj cpu credit pro pridelene casove okno (defaultne po 100ms) a po zbytek teto periody je zastaven. To muze napriklad zpusobit skokove casove rozdily u webovych requestu, kdy jeden probehne treba za 40ms, a dalsi za 120ms - protoze bude process cekat po vycerpani creditu. Jeste horsi situace muze nastat u nizkeho cpu limitu u vicethreadove aplikace, kdy kvota je vycerpana az rychlosti odpovidajici poctu threadu, takze treba 1cpu limit bude vycerpan 10ti thready ktere pouzivaji 100% cpu za 10ms a dalsich 90ms bude pod "odpocivat"

# Cast #2: Cpu throttling
Aby bylo mozne rozumne kontrolovat jak ktery pod vytezuje procesory na worker node, prideluje containerum v kubernetes podech urcity pocet cpu shares ktere muze vyuzit za 1s (cpu limit). Pod ovsem muze soucasne vyuzivat vice cpu (pokud ma vice threadu, nebo processu v containeru), takze za urcitych (a ne neobvyklych) okolnosti snadno dokaze spotrebovavat vice nez 1cpu/s. "Uctovani" cpu shares vykonava scheduler po takzvanych periodach, jez jsou v defaultnim nastaveni kubernetu dlouhe 100ms (je-li to nutne/vhodne, lze prekonfigurovat). Dojde-li k tomu, ze container predcasne vycerpa veskere sve cpu shares pro periodu, tak zbytek periody ceka. Napriklad pokud pridelime containeru 0.5cpu a uvnitr pobezi 2 thready/processy konzumujici 100% cpu kazdy, tak spotrebuji svuj "credit" zhruba za 25ms a nasledujicich 75ms nebudou delat nic. Toto je velmi nevhodne chovani pro aplikace citlive na odezvu a je nutne s tim pocitat pri sizingu containeru v podech.

Vytvoril jsem jednoduchou aplikaci, na niz je mozne demonstrovat toto chovani. Aplikace funguje tak, ze kazdou 1ms vypisuje na stdout cas od posledniho vypisu a vedle toho v n threadech (n je konfigurovatelne pri spusteni containeru) bezi nekonecna smycka konzumujici 100% cpu. Container/pod pak lze pustit s nastavenym cpu limitem a pozorovat k jak velkym casovym prodlevam dochazi po vycerpani cpu shares pro periodu. V aplikaci ve skutecnosti bezi n+1 threadu (aby thread pisici na stdout nebyl limitovan), takze vysledek neni uplne presny, ale pro demonstraci podle mne dostacujici.

Aplikace je dostupna zde: https://github.com/kacerr/blog/tree/main/001-kubernetes-requests-limits-throttling/throttling-tester. Jsem otevren jakymkoliv navrhum na vylepseni, protoze aplikaci jsem spise rychle spichl dohromady nez cokoliv jineho.

# Cast #3: Aplikace citlive na response time
Jak jiz s existence teto "kapitoly" vyplyva, univerzalni pravidlo pro nastaveni requestu a limitu neexistuje, ale asi se daji popsat nektere typy aplikaci sdilejici podobne charakteristiky a zkusit nalezt requesty/limity, ktere jim budou vyhovovat.

Konkretne u aplikaci, kde se ocekava rychlay a zaroven konzistentni response time je asi vhodne zvazit a nakombinovat vice moznosti:
- limity: ano/ne (nabizelo by se je uplne vypnout, ale to asi neni nejlepsi napad)
- provozovat aplikaci ve vice instancich: to je vhodne i z dalsich duvodu, ale z hlediska hledani konzistentiho vykonu asi potrebujeme pocet replik tunit vice nez treba pro nejaky "postupny" deployment nebo pro reseni HA v pripade nutnosti preschedulovat nejaky pod
- vzit v potaz design aplikace uvnitr podu: node aplikace zrejme vyuzije 1cpu pro event loopu a cpu bound funkce, ale bude-li v ni hodne async callu, tak dokaze vyuzite nejake dalsi cpucka (je vhodne asi experimentalne zjistit kolik a pak skalovat primarne pocet replik), naopak pustime-li treba nejakou go aplikaci s vysokym GOMAXPROCS a nastavime ji nejaky limit, tak dost snadnou muze dojit k tomu, ze rozpracuje 20 requestu soucasne a za 5ms vycerpa sve cpu shares, 95ms bude cekat a odpoved na request bude nesmyslne pomala

U techto typu aplikaci bych:
- Miril na to aby throttling byl nizky. Rekl bych, ze optimalne by se mel blizit 0, ale nejsem si jisty jestli je realne toho dosahnout bez uplneho vypnuti limitu.
- Hlavni metrikou zde bude konzistentni response time, ktery bude dosahovat vyhovujicich hodnot. Idelni je pripravit nejaky stress test s ocekavanym loadem a vyladit konfiguraci aplikace, limity, pocet replik tak abychom dokazali obslouzit co potrebujem

## jak merit throttling
Pokud mame posbirane metriky z cAdvisoru (zrejme ano, je soucasti kubeletu), tak mame k dispozici metriky:
- container_cpu_cfs_throttled_seconds_total
- container_cpu_cfs_throttled_periods_total

Metriku `container_cpu_cfs_throttled_seconds_total` chapu jako pocet vterin kdy container byl throttlovan == chtel neco delat, ale CFS scheduler ho nenechal.
`rate(container_cpu_cfs_throttled_seconds_total[1m])` nam tedy rika kolik vterin za vterinu container nemohl "neco delat, i kdyz chtel". Toto cislo muze byt bez problemu mnohem vetsi nez 1. Duvodem je prave to ze v containeru muze byt vice procesu/threadu, ktere jsou po vycerpani pridelenych cpu_shares blokovany.

## extras: prometheus queries
```
# throttling rate for pod
sum(rate(container_cpu_cfs_throttled_seconds_total{namespace="$namespace", pod=~"$pod", container!="", container!="POD"}[1m])) by (pod) or sum(rate(container_cpu_cfs_throttled_seconds_total{namespace="$namespace", pod_name=~"$pod", container_name!="", container_name!="POD"}[1m])) by (pod_name) 

# cpu usage for pod
sum(rate(container_cpu_usage_seconds_total{namespace="$namespace", pod=~"$pod", container!="", container!="POD"}[1m])) by (pod) or sum(rate(container_cpu_usage_seconds_total{namespace="$namespace", pod_name=~"$pod", container_name!="", container_name!="POD"}[1m])) by (pod_name)

# memory usage for pod
sum(container_memory_working_set_bytes{namespace="$namespace", pod="$pod", container!="", container!="POD", container=~"$containers"}) or sum(container_memory_working_set_bytes{namespace="$namespace", pod_name="$pod", container_name!="", container_name!="POD", container_name=~"$containers"})

```