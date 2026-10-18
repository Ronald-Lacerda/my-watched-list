# LESSONS - auto-maintained by scripts/lessons.py

> Machine-owned. Do NOT hand-edit. Changes are overwritten on the next `lessons.py` write.
> Canonical state lives in `.specs/lessons.json`. Edit lessons only via the script.
> promote_threshold=2 distinct features · window_days=45 · quarantine_threshold=2

## Confirmed (load these at Specify/Design)

Corroborated across multiple features. Safe to apply as guidance.

_none_

## Candidates (under observation - do NOT load as guidance yet)

Seen once or not yet corroborated. Tracked, not trusted.

### L-001 - Teste a rejeição de candidatos TMDB com nomes inexatos, além dos matches exatos.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `worker/tmdb` · harmful: 0
- features: fase-1
- evidence: M6 (worker/tmdb)
- last seen: 2026-10-18T21:58:34Z

### L-002 - Exija respostas de episódio com ID positivo e temporada e número iguais aos solicitados nos testes TMDB.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `worker/tmdb` · harmful: 0
- features: fase-1
- evidence: M7 (worker/tmdb)
- last seen: 2026-10-18T21:58:35Z

### L-003 - Nos testes Chromium, faça o servidor exigir o storage state recebido e confira o isolamento entre conexões.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `worker/netflix` · harmful: 0
- features: fase-1
- evidence: M8 (worker/netflix)
- last seen: 2026-10-18T21:58:36Z

### L-004 - Teste a execução inicial e recorrente do worker e seu encerramento por contexto, além do cálculo do cron.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `worker/syncer` · harmful: 0
- features: fase-1
- evidence: M9 (worker/syncer)
- last seen: 2026-10-18T21:58:36Z

### L-005 - Teste o despacho e as saídas dos comandos CLI válidos, além da validação de configuração.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `worker/cmd/worker` · harmful: 0
- features: fase-1
- evidence: M10 (worker/cmd/worker)
- last seen: 2026-10-18T21:58:37Z

### L-006 - Defina explicitamente a decisão de associação TMDB quando o tipo de mídia de origem é desconhecido.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `worker/tmdb` · harmful: 0
- features: fase-1
- evidence: spec.md:20 (worker/tmdb)
- last seen: 2026-10-18T21:58:38Z

### L-007 - Valide corpos opcionais de mutações com os mesmos limites e erros JSON dos demais endpoints.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `httpapi` · harmful: 0
- features: fase-2
- evidence: AC9/F1 (httpapi)
- last seen: 2026-10-18T23:02:51Z

### L-008 - Leia corpos de DELETE até EOF com limite, sem usar ContentLength positivo como prova de presença.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `httpapi` · harmful: 0
- features: fase-2
- evidence: AC14/F2 (httpapi)
- last seen: 2026-10-18T23:02:51Z

### L-009 - Asserte os campos e a orientação de reconexão renderizados no card, além do rótulo de estado.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `web` · harmful: 0
- features: fase-2
- evidence: M8/AC21 (web) (+1 more)
- last seen: 2026-10-18T23:09:37Z

## Quarantined (failed when applied - ignore)

A confirmed lesson that recurred alongside failure. Kept for the maintainer to review.

_none_
