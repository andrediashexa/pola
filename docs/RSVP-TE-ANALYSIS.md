# Análise: suporte a RSVP-TE no POLA

> Branch `feat/rsvp-te`. POLA (NTT) é um PCE stateful em Go que hoje fala **SR-TE
> e SRv6** via PCEP. Este doc analisa o que falta pra ele também controlar túneis
> **RSVP-TE** — motivado por uma rede Huawei NE8000 que é RSVP-TE (não SR), onde
> o PCEP em si já foi validado (com ODL) e sabemos exatamente o encoding que a
> caixa aceita.

## TL;DR — é viável e bem delimitado

PCEP é **agnóstico de sinalização** no nível de mensagem: PCInitiate/PCUpd/PCRpt
são os mesmos; o que muda entre SR e RSVP-TE é (1) o **tipo de subobjeto do ERO**
e (2) o **Path Setup Type (PST)** no objeto SRP/LSP. POLA já tem toda a máquina de
sessão, os objetos ENDPOINTS e BANDWIDTH, e até a constante `PathSetupTypeRSVPTE
= 0x00`. Falta principalmente **encodar o ERO com subobjetos IPv4 (RFC 3209)** e
ligar o PST=0.

**Vantagem que já conhecemos:** a NE8000 só sinaliza ERO **loose** de loopbacks
(strict falha com RSVP 24/5). POLA mandando subobjeto IPv4 com L-flag=1 reproduz
exatamente o que funciona — sem o peso do ODL/Java.

## O que o POLA já tem (reaproveitável)

| Peça | Onde | Serve pra RSVP-TE? |
|---|---|---|
| Sessão PCEP (Open/Keepalive/FSM) | `pkg/server/session.go`, `server.go` | ✅ igual |
| PCInitiate / PCUpd / PCRpt | `pkg/packet/pcep/message.go` | ✅ estrutura igual |
| Objeto **ENDPOINTS** (src/dst) | `object.go` (`EndpointsObject`) | ✅ RSVP precisa |
| Objeto **BANDWIDTH** | `object.go` (`BandwidthObject`) | ✅ reserva TE |
| Objeto **ERO** (container) | `object.go` (`EroObject`) | ✅ só faltam subobjetos IPv4 |
| Objeto **LSP** + **SRP** | `object.go` | ✅ só falta branch PST |
| **`PathSetupTypeRSVPTE = 0x00`** | `tlv.go:808` | ✅ **já definido**, não usado |
| TED via BGP-LS (GoBGP) | `internal/pkg/gobgp`, `pkg/table/ted.go` | ✅ (links já têm IPs de interface) |
| CSPF | `pkg/cspf/cspf.go` | ⚠️ devolve segmentos SR; p/ RSVP devolveria hops IPv4 |
| gRPC API + CLI | `api/pola/v1`, `cmd/pola/sr_policy_*` | ⚠️ precisa um par "rsvp-tunnel" |

## O que falta (file-by-file)

### 1. Subobjeto ERO IPv4 — RFC 3209 §4.3.3.1 · `pkg/packet/pcep/object.go`
Hoje `EroObject.DecodeFromBytes` só conhece `SubObjectTypeEROSR (0x24)` e SRv6.
Adicionar o subobjeto **IPv4 prefix (type 0x01)**:

```
0               1               2               3
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|L|    Type=1  |     Length=8  |          IPv4 address ...     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| ... IPv4 address (cont.)      | Prefix Length |   Reserved    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```
- byte0 = `L<<7 | 0x01` (L=1 ⇒ **loose** — o que a NE8000 quer)
- byte1 = length = 8
- bytes2-5 = IPv4 (4 octetos)
- byte6 = prefix-length = 32
- byte7 = reservado (0)

Implementar uma struct `IPv4EroSubobject` que satisfaça a interface `EroSubobject`
(`DecodeFromBytes`/`Len`/`Serialize`/`ToSegment`) + um `case 0x01` no decode. (Há
ainda RFC 3209 type 32 = AS-number e unnumbered, opcionais — pular no MVP.)

### 2. Tipo de segmento RSVP · `pkg/table/sr_policy.go` (ou novo `rsvp.go`)
`table.Segment` é a interface comum. Criar `SegmentRSVPIPv4 { Addr netip.Addr;
Loose bool }` implementando os métodos. `NewEroSubobject(seg)` (object.go) ganha
um `case table.SegmentRSVPIPv4 → NewIPv4EroSubObject`.

### 3. PST=0 no SRP/LSP · `object.go`
`NewSrpObject` decide o PST pelo tipo do 1º segmento (SRMPLS→SRTE, SRv6→SRv6TE).
Adicionar branch: `SegmentRSVPIPv4 → PathSetupType{PathSetupTypeRSVPTE}` (ou
**omitir** a TLV — RSVP é o default; testar o que a NE8000 prefere). Idem no LSP
object onde aplicável.

### 4. Construtores de mensagem · `message.go`
`NewPCInitiateMessage`/`NewPCUpdMessage` recebem `segmentList []table.Segment` —
já genérico. Garantir que, com `SegmentRSVPIPv4`, eles:
- montem o ERO IPv4 (via 1+2),
- incluam **ENDPOINTS** (src/dst) — obrigatório no RSVP PCInitiate,
- opcional **BANDWIDTH** (reserva),
- PST correto (via 3).
Provavelmente um parâmetro/opção `pathType` ou inferência pelo segmento.

### 5. Modelo + API + CLI
- `pkg/table`: um `RsvpTunnel { Name, Src, Dst, Hops []SegmentRSVPIPv4, Bandwidth }`
  paralelo a `SRPolicy` (ou um campo `PathSetupType` no modelo existente).
- `api/pola/v1`: rpc `CreateRsvpTunnel`/`DeleteRsvpTunnel`/`GetRsvpTunnelList`
  (espelhar os de SR policy no `.proto` + regenerar).
- `cmd/pola`: `rsvp-tunnel add/list/delete` (espelhar `sr_policy_*`).

### 6. CSPF (opcional no MVP) · `pkg/cspf/cspf.go`
Hoje devolve `[]Segment` SR. Pra RSVP, ou (a) o usuário passa **hops explícitos
loose** (MVP — e é o que a NE8000 quer mesmo), ou (b) estende o CSPF pra devolver
hops IPv4 a partir da TED (os links já carregam IPs de interface). MVP: hops
explícitos; CSPF depois.

## Esforço estimado

| Fase | Conteúdo | Tamanho |
|---|---|---|
| **F1** | Subobjeto ERO IPv4 + SegmentRSVP + PST + testes de encode | ~1-2 dias |
| **F2** | Construtores PCInitiate/PCUpd RSVP (ERO+ENDPOINTS+BW+PST) | ~1 dia |
| **F3** | Modelo + gRPC + CLI `rsvp-tunnel` | ~1-2 dias |
| **F4** | Teste ao vivo contra a NE8000 (loose), iterar | ~1 dia |
| **F5** | CSPF IPv4 (opcional) | ~2 dias |

MVP útil (F1-F4) ≈ **4-6 dias**. PCEP já foi provado com a caixa (loose IPv4 ERO);
o risco é encoding/interop, não conceito.

## Riscos / notas
- **Loose vs strict:** mandar **loose** (L=1). Strict de loopback falha na NE8000
  (validado na saga com ODL: RSVP Error 24/5).
- **PST omit vs explicit 0:** algumas caixas querem a TLV PST=0, outras assumem
  RSVP por ausência. Testar ambos via debug PCEP na caixa.
- **Delegação:** RSVP-TE PCUpd só modifica CR-LSP **active-delegate + já UP**;
  PCInitiate cria. Mesma matriz de delegação do ODL.
- **AO/PPAG:** se a caixa usa hot-standby, o encoding de associação (RFC 8697/8745)
  importa (`ppag standard enable` no Huawei). POLA tem objeto de associação?
  Verificar `AssociationObject` em message.go (aparece no PCInitiate).
- **Capabilities no Open:** a caixa anuncia `stateful` + `initiated-lsp`; POLA já
  negocia stateful. RSVP não exige capability SR — então até simplifica.

## Por que vale (vs ODL)
- POLA = **um binário Go leve**, sem JVM/Karaf/feature-install/autoheal.
- Controla RSVP-TE loose — exatamente o que a NE8000 aceita — sem a stack ODL.
- TED via GoBGP (o mesmo que já usamos).
- Roadmap natural: depois agrega o SR/SRv6 nativo do POLA se a rede migrar.

## Progresso nesta branch

- **F1 ✅** — `SegmentRSVPIPv4` (`pkg/table/rsvp.go`) + `IPv4EroSubobject` (RFC
  3209, `pkg/packet/pcep/ero_ipv4.go`, L-flag=loose) + decode `case 0x01` +
  `NewEroSubobject`/`NewSrpObject` (PST=RSVPTE). Testes: serialize loose=`0x81`/
  strict=`0x01`, round-trip, ERO completo.
- **F2 ✅** — construtores `NewPCInitiateMessageRSVP` / `NewPCUpdMessageRSVP`
  (`pkg/packet/pcep/message_rsvp.go`): SRP(PST=0) + LSP + ENDPOINTS + ERO IPv4
  loose, **sem** ASSOCIATION/VENDOR (esses são SR-policy). BANDWIDTH pulado no
  MVP (loose sem reserva, igual o fluxo loose validado com ODL). Testes de
  estrutura + serialização. `go build ./...` OK.

**Próximo: F3** — modelo `RsvpTunnel` + gRPC (`.proto` add/list/delete) + CLI
`rsvp-tunnel`, ligando os construtores acima na sessão (`pkg/server/session.go`).
Depois **F4** — teste ao vivo contra a NE8000 da SOS (apontar `connect-server`
da caixa pro POLA; POLA pode rodar paralelo ao ODL). Mandar **loose**.
