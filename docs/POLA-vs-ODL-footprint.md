# Consumo de hardware: POLA vs OpenDaylight (PCE)

Medições ao vivo na VM (62 GB RAM, 2026-06-20). Caso de uso: **PCE pra controlar
túneis TE numa rede Huawei** (PCEP + TED via BGP-LS).

## Números medidos

| Métrica | **POLA** (Go) | **OpenDaylight** (JVM/Karaf) | Razão |
|---|---|---|---|
| **RAM (RSS) idle** | **13,4 MB** | **~1,5 GiB** (1591 MiB) | **~115× menos** |
| Heap máx configurado | n/a (GC do Go) | `JAVA_MAX_MEM=4g` | — |
| **Binário / imagem** | **13 MB** (binário estático) | **1,29 GB** (imagem docker) | ~100× menor |
| Threads idle | **6** | centenas (JVM+OSGi+akka) | — |
| CPU idle | ~1% | ~1% | ≈ |
| Boot até pronto | **< 1 s** | **~60-90 s** (Karaf + features + RESTCONF) | dezenas× |
| Runtime | binário único, **sem deps** | precisa **JDK 17** + Karaf/OSGi | — |

> POLA medido idle (PCEP + gRPC up, sem peer/TED). Com TED (BGP-LS via GoBGP) sobe
> alguns MB — a TED de 24 nós/54 links é trivial em memória; mesmo 670 nós cabe em
> dezenas de MB (grafo Go). ODL com a mesma TED consome mais que o idle medido.

## Por que a diferença

- **POLA**: binário Go estático, GC próprio, escopo enxuto (PCE stateful + TED).
  Sem JVM, sem OSGi, sem datastore clusterizado.
- **ODL**: JVM (JDK 17) + Apache Karaf (OSGi) + MD-SAL datastore + akka cluster +
  bgpcep com **todos** os SAFIs/algo/RESTCONF. Muito mais máquina rodando.

## Leitura honesta (apples-to-oranges)

ODL faz **muito mais** que um PCE: RESTCONF northbound, datastore YANG,
clustering, BGP completo (várias famílias), flowspec, BMP, etc. A comparação é
justa **só pro recorte "PCE pra TE"** — que é o nosso uso. Pra esse recorte:

- **POLA** = ~115× menos RAM, ~100× menor em disco, boot instantâneo, 1 binário.
- **ODL** = stack pesada, mas traz junto coisas que o POLA não tem (RESTCONF,
  flowspec, BMP, multi-SAFI, computação algo madura).

## Implicação prática

Pra rodar **N PCEs** (ex. 1 por provedor, multi-tenant) ou em hardware modesto,
POLA é ordens de grandeza mais barato: 10 instâncias POLA ≈ 130 MB; 10 ODLs ≈ 15
GB. O custo é desenvolver o que falta (ex. **RSVP-TE**, ver `RSVP-TE-ANALYSIS.md`)
— mas o PCEP base já está provado contra a NE8000.
