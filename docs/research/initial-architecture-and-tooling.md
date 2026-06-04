# Исходные архитектурные и tooling-ориентиры

Question: Какие современные первичные источники поддерживают или опровергают
исходные решения Intercessio?

## Findings

### Go interfaces

Официальные Go Code Review Comments рекомендуют определять интерфейсы в package,
который их использует, возвращать concrete types со стороны реализации и не
создавать интерфейс до реалистичного примера использования. Это напрямую
противоречит буквальному «сначала перечислим все интерфейсы», но совместимо с
hexagonal architecture, если port появляется из use case.

### LLM authority

OWASP LLM06:2025 связывает ущерб agentic-систем с excessive functionality,
permissions и autonomy. В числе mitigations — минимальные capabilities,
downstream authorization и независимая проверка high-impact actions. NIST GenAI
Profile рекомендует risk tiering, дополнительный human review, tracking,
documentation и change-management controls.

Это поддерживает предлагаемое разделение: LLM предлагает, policy разрешает,
владелец состояния исполняет.

### Agent memory and hooks

Serena позиционируется прежде всего как symbol-aware MCP toolkit с LSP/JetBrains
backend; memory можно отключить. Поэтому она дополняет, но не должна заменять
tool-neutral project knowledge.

Lefthook поддерживает tracked commands с glob/file filters и подходит для
одинакового локального и CI entry point. Hook может доказать только механические
свойства; documentation meaning остается review concern.

### Architecture and event catalogs

LikeC4 дает architecture-as-code DSL и генерируемые views. AsyncAPI описывает
protocol-agnostic external message APIs. EventCatalog связывает domains,
systems, messages, schemas, flows и ADR. Все три полезны после появления
стабильных объектов модели; до этого они рискуют формализовать гипотезу.

## Sources

- [Go Code Review Comments: Interfaces](https://go.dev/wiki/CodeReviewComments#interfaces)
- [OWASP LLM06:2025 Excessive Agency](https://genai.owasp.org/llmrisk/llm062025-excessive-agency/)
- [NIST AI RMF Generative AI Profile](https://nvlpubs.nist.gov/nistpubs/ai/NIST.AI.600-1.pdf)
- [Serena](https://github.com/oraios/serena)
- [Lefthook commands](https://lefthook.dev/configuration/Commands/)
- [LikeC4](https://likec4.dev/)
- [AsyncAPI documentation](https://www.asyncapi.com/docs)
- [EventCatalog](https://www.eventcatalog.dev/)

## Applicability to Intercessio

Источники подтверждают технический baseline, но не определяют нормативную модель
вымышленной политии. Внешняя рекомендация не заменяет решение автора.

## Unknowns and expiry trigger

Повторно проверить agent tooling перед установкой и AI security guidance перед
первым live LLM integration. Исторические и институциональные источники
исследовать после выбора первого scenario; они не становятся нормами симуляции
без отдельного решения.
