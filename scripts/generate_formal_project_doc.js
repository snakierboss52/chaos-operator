const fs = require("fs");
const {
  AlignmentType,
  BorderStyle,
  Document,
  Footer,
  Header,
  HeadingLevel,
  LevelFormat,
  PageBreak,
  PageNumber,
  Paragraph,
  Packer,
  ShadingType,
  Table,
  TableCell,
  TableRow,
  TextRun,
  VerticalAlign,
  WidthType,
} = require("docx");

const OUTPUT = process.argv[2] || "Documento_Formal_Proyecto_Operador_Ingenieria_Caos.docx";

const COLORS = {
  navy: "203864",
  blue: "2F5597",
  lightBlue: "D9EAF7",
  paleBlue: "EAF2F8",
  gray: "5B6573",
  lightGray: "F2F3F5",
  border: "B8C2CC",
  white: "FFFFFF",
  green: "E2F0D9",
  greenText: "375623",
  amber: "FFF2CC",
  amberText: "7F6000",
  red: "FCE4D6",
  redText: "9C0006",
  purple: "E4DFEC",
  purpleText: "4C3B6E",
};

const PAGE_WIDTH = 11906;
const PAGE_HEIGHT = 16838;
const MARGIN = 1134;
const CONTENT_WIDTH = PAGE_WIDTH - 2 * MARGIN;

const borders = {
  top: { style: BorderStyle.SINGLE, size: 4, color: COLORS.border },
  bottom: { style: BorderStyle.SINGLE, size: 4, color: COLORS.border },
  left: { style: BorderStyle.SINGLE, size: 4, color: COLORS.border },
  right: { style: BorderStyle.SINGLE, size: 4, color: COLORS.border },
  insideHorizontal: { style: BorderStyle.SINGLE, size: 3, color: "D9DEE3" },
  insideVertical: { style: BorderStyle.SINGLE, size: 3, color: "D9DEE3" },
};

const numbering = {
  config: [
    {
      reference: "bullets",
      levels: [
        {
          level: 0,
          format: LevelFormat.BULLET,
          text: "•",
          alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 540, hanging: 260 } } },
        },
        {
          level: 1,
          format: LevelFormat.BULLET,
          text: "–",
          alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 980, hanging: 260 } } },
        },
      ],
    },
    {
      reference: "steps",
      levels: [
        {
          level: 0,
          format: LevelFormat.DECIMAL,
          text: "%1.",
          alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 620, hanging: 320 } } },
        },
      ],
    },
    {
      reference: "stepsDesign",
      levels: [
        {
          level: 0,
          format: LevelFormat.DECIMAL,
          text: "%1.",
          alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 620, hanging: 320 } } },
        },
      ],
    },
  ],
};

function t(text, opts = {}) {
  return new TextRun({
    text,
    font: opts.font || "Aptos",
    size: opts.size || 21,
    bold: opts.bold || false,
    italics: opts.italics || false,
    color: opts.color || "20242A",
    break: opts.break,
  });
}

function p(text, opts = {}) {
  const children = Array.isArray(text) ? text : [t(text, opts)];
  return new Paragraph({
    children,
    alignment: opts.alignment || AlignmentType.JUSTIFIED,
    spacing: { before: opts.before || 0, after: opts.after ?? 110, line: opts.line || 276 },
    indent: opts.indent,
    keepNext: opts.keepNext,
    pageBreakBefore: opts.pageBreakBefore,
  });
}

function bullet(text, level = 0) {
  return new Paragraph({
    children: [t(text)],
    numbering: { reference: "bullets", level },
    spacing: { after: 60, line: 260 },
  });
}

function step(text, reference = "steps") {
  return new Paragraph({
    children: [t(text)],
    numbering: { reference, level: 0 },
    spacing: { after: 80, line: 260 },
  });
}

function h1(text, pageBreakBefore = true) {
  return new Paragraph({
    text,
    heading: HeadingLevel.HEADING_1,
    pageBreakBefore,
    spacing: { before: 0, after: 180 },
    keepNext: true,
  });
}

function h2(text) {
  return new Paragraph({
    text,
    heading: HeadingLevel.HEADING_2,
    spacing: { before: 240, after: 100 },
    keepNext: true,
  });
}

function h3(text) {
  return new Paragraph({
    text,
    heading: HeadingLevel.HEADING_3,
    spacing: { before: 160, after: 70 },
    keepNext: true,
  });
}

function note(title, body, color = COLORS.paleBlue) {
  return new Table({
    width: { size: CONTENT_WIDTH, type: WidthType.DXA },
    columnWidths: [CONTENT_WIDTH],
    borders,
    rows: [
      new TableRow({
        children: [
          new TableCell({
            width: { size: CONTENT_WIDTH, type: WidthType.DXA },
            shading: { type: ShadingType.CLEAR, fill: color },
            margins: { top: 120, bottom: 120, left: 160, right: 160 },
            children: [
              p([t(title + ". ", { bold: true, color: COLORS.navy }), t(body)], { after: 0 }),
            ],
          }),
        ],
      }),
    ],
  });
}

function cell(text, width, opts = {}) {
  const paragraphs = [];
  const lines = Array.isArray(text) ? text : [text];
  for (const line of lines) {
    if (line instanceof Paragraph) {
      paragraphs.push(line);
    } else {
      paragraphs.push(
        p(String(line), {
          alignment: opts.alignment || AlignmentType.LEFT,
          after: opts.after ?? 30,
          line: 235,
          bold: opts.bold,
          color: opts.color,
          size: opts.size || 18,
        }),
      );
    }
  }
  return new TableCell({
    width: { size: width, type: WidthType.DXA },
    verticalAlign: VerticalAlign.CENTER,
    shading: opts.fill ? { type: ShadingType.CLEAR, fill: opts.fill } : undefined,
    margins: { top: 90, bottom: 90, left: 100, right: 100 },
    children: paragraphs,
  });
}

function table(headers, rows, widths, opts = {}) {
  const header = new TableRow({
    tableHeader: true,
    cantSplit: true,
    children: headers.map((x, i) =>
      cell(x, widths[i], {
        bold: true,
        color: COLORS.white,
        fill: opts.headerFill || COLORS.navy,
        alignment: opts.headerAlign || AlignmentType.CENTER,
        size: 18,
      }),
    ),
  });
  const body = rows.map(
    (row, ri) =>
      new TableRow({
        cantSplit: true,
        children: row.map((x, i) =>
          cell(x, widths[i], {
            fill: ri % 2 === 1 ? COLORS.lightGray : COLORS.white,
            size: opts.bodySize || 18,
          }),
        ),
      }),
  );
  return new Table({
    width: { size: widths.reduce((a, b) => a + b, 0), type: WidthType.DXA },
    columnWidths: widths,
    borders,
    rows: [header, ...body],
  });
}

function statusTable() {
  const w = Math.floor(CONTENT_WIDTH / 4);
  return new Table({
    width: { size: w * 4, type: WidthType.DXA },
    columnWidths: [w, w, w, w],
    borders,
    rows: [
      new TableRow({
        children: [
          cell("VERIFICADO\nExiste evidencia directa en código o prueba ejecutada.", w, { fill: COLORS.green, bold: true, color: COLORS.greenText, size: 17 }),
          cell("PARCIAL\nExiste una base, pero faltan comportamientos o validación.", w, { fill: COLORS.amber, bold: true, color: COLORS.amberText, size: 17 }),
          cell("PENDIENTE\nNo se encontró implementado o no hay evidencia suficiente.", w, { fill: COLORS.red, bold: true, color: COLORS.redText, size: 17 }),
          cell("PROPUESTO\nDecisión razonada que requiere aprobación del proyecto.", w, { fill: COLORS.purple, bold: true, color: COLORS.purpleText, size: 17 }),
        ],
      }),
    ],
  });
}

function spacer(height = 100) {
  return new Paragraph({ spacing: { before: height, after: 0 } });
}

const functionalRows = [
  ["RF-01", "Definir los CRD PodChaos, NetworkChaos, StressChaos y HTTPChaos bajo chaos.engineering.io/v1alpha1.", "Verificado", "api/v1alpha1 y config/crd"],
  ["RF-02", "Permitir crear experimentos mediante manifiestos YAML y reconciliarlos desde la API de Kubernetes.", "Verificado", "Cuatro reconcilers registrados en internal/app/app.go"],
  ["RF-03", "Seleccionar pods por namespace, etiquetas, campos, fase o lista explícita.", "Parcial", "Los filtros principales existen; nodeSelectors no se aplican y la consulta con múltiples namespaces parte del primero."],
  ["RF-04", "Soportar modos one, all, fixed, fixed-percent y random-max-percent.", "Verificado", "internal/domain/target_selector.go y pruebas unitarias"],
  ["RF-05", "Limitar objetivos con maxPods y maxPercentage; aplicar un límite por defecto.", "Verificado", "BlastRadiusControl y TargetSelector"],
  ["RF-06", "Ejecutar PodChaos: pod-kill, pod-failure y container-kill.", "Parcial", "Las acciones se enrutan; pod-failure termina el pod y force no se utiliza."],
  ["RF-07", "Ejecutar NetworkChaos: delay, loss, duplicate, corrupt, partition y bandwidth.", "Parcial", "Se generan reglas tc/iptables; direction, target y externalTargets no delimitan la partición actual."],
  ["RF-08", "Ejecutar StressChaos sobre CPU, memoria, disco e I/O, de forma individual o combinada.", "Parcial", "Los scripts existen; load, options y oomScoreAdj no se aplican."],
  ["RF-09", "Ejecutar HTTPChaos con abort, delay, replace y patch.", "Parcial", "Abort y replace alteran Nginx; delay no introduce espera real y patch no se construye."],
  ["RF-10", "Gestionar el ciclo Pending → Running → Completed/Failed y registrar tiempos y condiciones.", "Verificado", "Estados y Conditions implementados en los reconcilers"],
  ["RF-11", "Recuperar efectos reversibles al vencer duration o al eliminar el recurso.", "Verificado", "Finalizers y cleanup para NetworkChaos, StressChaos y HTTPChaos"],
  ["RF-12", "Registrar pods afectados y resultados por objetivo.", "Verificado", "ChaosStatus, ExperimentResult y TargetResult"],
  ["RF-13", "Emitir eventos de Kubernetes, métricas Prometheus y un resumen por experimento.", "Verificado", "EventRecorder, metrics.go y ConfigMap chaos-report-*"],
  ["RF-14", "Aplicar defaults y rechazar especificaciones inválidas mediante admission webhooks.", "Parcial", "Código y manifiestos existen; ENABLE_WEBHOOKS=false por defecto y no se evidenció aprovisionamiento TLS."],
  ["RF-15", "Permitir ondas repetidas durante un experimento.", "Verificado", "Anotación chaos.engineering.io/wave-interval en Pod, Network y HTTP"],
  ["RF-16", "Programar experimentos con expresiones cron.", "Pendiente", "SchedulerSpec existe, pero no hay controlador ni validación/ejecución cron."],
  ["RF-17", "Abortar automáticamente por umbrales de métricas o disponibilidad.", "Pendiente", "Descrito en el documento original, no encontrado en el código."],
  ["RF-18", "Restringir namespaces permitidos/protegidos desde configuración.", "Pendiente", "No existe configuración equivalente ni control previo a la ejecución."],
];

const nonFunctionalRows = [
  ["RNF-01", "Seguridad", "Ejecutar con ServiceAccount dedicado, usuario no root, filesystem de solo lectura y sin capacidades Linux.", "Verificado", "config/manager/deployment.yaml"],
  ["RNF-02", "Mínimo privilegio", "Limitar RBAC a las operaciones indispensables y revisar permisos de create/update/patch/delete sobre pods.", "Parcial", "ClusterRole dedicado, con permisos más amplios que los usados por todos los flujos."],
  ["RNF-03", "Validación", "Aceptar únicamente valores y formatos permitidos antes de construir comandos tc, iptables, dd o Nginx.", "Pendiente crítico", "Varios valores del CR se interpolan en sh -c; se requieren allowlists estrictas y tipos validados."],
  ["RNF-04", "Contención", "Impedir objetivos en namespaces protegidos y etiquetas reservadas, además del blast radius.", "Pendiente", "El límite cuantitativo existe; la exclusión cualitativa no."],
  ["RNF-05", "Recuperación", "Restituir únicamente los cambios creados por el experimento, incluso ante eliminación o reinicio.", "Parcial", "Hay finalizers; NetworkChaos limpia cadenas completas de iptables, lo que puede remover reglas ajenas."],
  ["RNF-06", "Observabilidad", "Exponer métricas, health/readiness, eventos, estados y reportes consultables.", "Verificado", "Prometheus, /healthz, /readyz, Events y ConfigMaps"],
  ["RNF-07", "Rendimiento", "Cumplir los SLO documentados para smoke, steady y spike.", "Pendiente de evidencia", "loadtest/ define umbrales; no se ejecutaron en este análisis."],
  ["RNF-08", "Mantenibilidad", "Mantener capas separadas, pruebas automatizadas y cobertura acordada.", "Parcial", "Pruebas pasan; cobertura global observada: 13,3 %. Meta formal por aprobar."],
  ["RNF-09", "Disponibilidad", "Permitir leader election y definir estrategia de alta disponibilidad.", "Parcial", "Leader election está activa, pero el Deployment usa una sola réplica."],
  ["RNF-10", "Compatibilidad", "Declarar y validar versiones soportadas de Kubernetes, runtime, herramientas en pods y Nginx.", "Pendiente", "Go 1.22.5 y client-go v0.29.0 están fijados; el rango de compatibilidad no está demostrado."],
  ["RNF-11", "Portabilidad", "Construir imagen y desplegar de forma reproducible en un clúster kind.", "Verificado para desarrollo", "Dockerfile, Makefile, kind-cluster.yaml, install.sh"],
  ["RNF-12", "Aleatoriedad", "Seleccionar objetivos sin usar math/rand y definir si se requiere reproducibilidad por semilla.", "Pendiente de decisión", "El código usa math/rand; debe acordarse comportamiento y reemplazo seguro."],
];

const implementationRows = [
  ["Entrada y arranque", "cmd/manager; internal/app", "Inicializa scheme, manager, logging, health checks, leader election y cuatro controladores.", "Implementado"],
  ["Modelo de API", "api/v1alpha1", "Tipos, status compartido, modos, selectores, blast radius y webhooks.", "Implementado con brechas de validación"],
  ["Controladores", "controllers", "Ciclo de reconciliación, finalizers, ejecución, monitoreo, recuperación, métricas y eventos.", "Implementado"],
  ["Dominio", "internal/domain", "Interfaces ChaosExecutor/operaciones y selección de objetivos.", "Implementado; validaciones de executor incompletas"],
  ["Infraestructura", "internal/infrastructure", "Cliente Kubernetes, pods/exec, scripts de falla, métricas y reportes.", "Implementado; requiere endurecimiento de inputs"],
  ["Despliegue", "config/manager; config/rbac; config/crd", "Deployment, ServiceAccount, ClusterRole, Service y CRD.", "Implementado"],
  ["Webhooks", "config/webhook; api/v1alpha1/*_webhook.go", "Defaulting y validación de admisión.", "Parcial: activación/TLS no demostrados"],
  ["Observabilidad", "monitoring; metrics.go", "Prometheus, Grafana y dashboards del proyecto.", "Artefactos presentes; despliegue no validado en esta revisión"],
  ["Pruebas de carga", "loadtest", "k6/xk6-kubernetes: smoke, steady y spike; reportes y umbrales.", "Artefactos presentes; resultados no adjuntos"],
];

const differenceRows = [
  ["Modelo de recursos", "ChaosExperiment, ChaosSchedule y ChaosResult", "PodChaos, NetworkChaos, StressChaos y HTTPChaos", "Adoptar el código como fuente vigente."],
  ["API group", "chaos.operator.io/v1alpha1", "chaos.engineering.io/v1alpha1", "Corregir toda documentación y ejemplos."],
  ["Programación", "ChaosSchedule con cron y políticas de concurrencia", "Campo scheduler.cron sin lógica de ejecución", "Definir si se implementa o se retira del alcance."],
  ["Resultados", "CRD ChaosResult independiente", "Status embebido y ConfigMap chaos-report-*", "Documentar el mecanismo real."],
  ["Abortos", "Umbrales Prometheus, disponibilidad y rollback", "No implementados", "Decidir criterios y diseño antes de producción."],
  ["Restricciones", "Namespaces permitidos/protegidos por ConfigMap", "No implementadas", "Tratar como requisito de seguridad pendiente."],
  ["Tipos adicionales", "IOChaos y TimeChaos aparecen en documentación secundaria", "No hay CRD/controlador propio", "Excluir de esta versión."],
];

const testRows = [
  ["CP-01", "Automática", "Ejecutar go test ./... y registrar cobertura.", "Todos los paquetes pasan; cobertura queda registrada.", "Salida de consola + cover.out"],
  ["CP-02", "Integración", "Instalar los cuatro CRD en kind y crear un recurso válido de cada tipo.", "CRD aceptados y status deja de estar vacío.", "kubectl get/describe + eventos"],
  ["CP-03", "Negativa", "Con webhooks activos, enviar duration, mode/value y fault specs inválidos.", "Admission rechaza cada manifiesto antes de la ejecución.", "Mensaje de kubectl + logs webhook"],
  ["CP-04", "E2E", "Aplicar PodChaos pod-kill sobre un Deployment con varias réplicas y maxPods=1.", "Se elimina como máximo un pod; el controlador registra Running/Completed y Kubernetes repone la réplica.", "Capturas de pods, CR, eventos y métricas"],
  ["CP-05", "E2E", "Aplicar container-kill indicando un contenedor válido.", "El PID 1 termina, el contenedor reinicia y el CR registra el objetivo.", "restartCount, describe y logs"],
  ["CP-06", "E2E", "Aplicar NetworkChaos delay/loss y esperar duration.", "tc muestra la regla durante la prueba y queda sin la regla del experimento al terminar.", "tc qdisc antes/durante/después"],
  ["CP-07", "Seguridad/E2E", "Aplicar network-partition y revisar la recuperación.", "La partición se limita al objetivo y la limpieza no elimina reglas ajenas.", "iptables antes/durante/después; hoy requiere validación cuidadosa"],
  ["CP-08", "E2E", "Ejecutar StressChaos combinado (CPU, memoria, disco, I/O) con duración corta.", "Se crean procesos/archivos previstos y todos se eliminan al completar o borrar el CR.", "ps, top, df, archivos y annotations"],
  ["CP-09", "E2E", "Aplicar HTTPChaos abort y replace sobre el workload Nginx de prueba.", "La respuesta cambia durante el experimento y se restaura luego.", "curl antes/durante/después + config Nginx"],
  ["CP-10", "Brecha conocida", "Aplicar HTTPChaos delay y medir latencia real.", "La latencia aumenta según spec; la versión actual probablemente no cumple porque solo agrega headers/respuesta.", "curl -w, histograma o k6"],
  ["CP-11", "Funcional", "Probar one/all/fixed/fixed-percent/random-max-percent con maxPods y maxPercentage.", "Ninguna ejecución supera el menor límite configurado y el conjunto cumple el modo.", "Lista de candidatos, AffectedPods y Result"],
  ["CP-12", "Observabilidad", "Completar un experimento y consultar métricas, eventos y chaos-report-*.", "Contadores, tiempos, objetivos y fase coinciden entre las tres fuentes.", "PromQL, kubectl events y ConfigMap"],
  ["CP-13", "Recuperación", "Eliminar NetworkChaos/StressChaos/HTTPChaos durante Running.", "El finalizer ejecuta cleanup y luego permite eliminar el CR.", "timestamps, eventos, estado del pod"],
  ["CP-14", "Seguridad", "Probar payloads con metacaracteres de shell/Nginx en latencia, rate, size, path, headers y body.", "Todos son rechazados por validación; nunca llegan a sh -c.", "Respuesta admission + logs; bloqueante si falla"],
  ["CP-15", "Seguridad", "Intentar seleccionar kube-system, chaos-system y un pod marcado como protegido.", "El operador rechaza o excluye esos objetivos.", "Manifiesto y salida; requisito aún no implementado"],
  ["CP-16", "Carga", "Ejecutar smoke, steady y spike definidos en loadtest/.", "Cumple los thresholds documentados y no deja CRs bloqueados ni reinicios del operador.", "Resumen k6, Prometheus, Grafana y reporte generado"],
];

const docChildren = [];

// Cover
docChildren.push(
  spacer(900),
  new Paragraph({
    children: [t("OPERADOR DE INGENIERÍA DEL CAOS PARA KUBERNETES", { bold: true, size: 38, color: COLORS.navy })],
    alignment: AlignmentType.CENTER,
    spacing: { after: 260 },
  }),
  new Paragraph({
    children: [t("Documento técnico de formulación y estado de implementación", { size: 28, color: COLORS.blue })],
    alignment: AlignmentType.CENTER,
    spacing: { after: 650 },
  }),
  new Paragraph({
    children: [t("Proyecto Práctica IV", { bold: true, size: 24, color: COLORS.gray })],
    alignment: AlignmentType.CENTER,
    spacing: { after: 90 },
  }),
  new Paragraph({
    children: [t("Jorge Humberto Lozano Arenas", { size: 23 })],
    alignment: AlignmentType.CENTER,
    spacing: { after: 70 },
  }),
  new Paragraph({
    children: [t("Universidad Central — Ingeniería de Sistemas", { size: 22 })],
    alignment: AlignmentType.CENTER,
    spacing: { after: 620 },
  }),
  new Paragraph({
    children: [t("Versión 1.0  |  20 de agosto de 2026", { size: 20, color: COLORS.gray })],
    alignment: AlignmentType.CENTER,
    spacing: { after: 140 },
  }),
  new Paragraph({
    children: [t("Documento nuevo. El archivo fuente suministrado no fue modificado.", { italics: true, size: 19, color: COLORS.gray })],
    alignment: AlignmentType.CENTER,
  }),
  new Paragraph({ children: [new PageBreak()] }),
);

// Editorial note and contents
docChildren.push(
  new Paragraph({ text: "Nota de elaboración", heading: HeadingLevel.TITLE, spacing: { after: 180 } }),
  p("Este documento reorganiza y depura el contenido del archivo suministrado y lo contrasta con el repositorio goland-operator. El código fuente se considera la evidencia principal para afirmar que una capacidad está implementada. Las ideas presentes solo en documentos se conservan como propuestas o pendientes, sin presentarlas como hechos."),
  p("Fuentes revisadas: documento académico suministrado; código Go, CRD, manifiestos, muestras, documentación y pruebas del repositorio; ejecución local de la suite automatizada el 20 de agosto de 2026."),
  spacer(80),
  statusTable(),
  spacer(240),
  new Paragraph({ text: "Contenido", heading: HeadingLevel.TITLE, spacing: { after: 160 } }),
  ...[
    "1. Nombre del proyecto",
    "2. Objetivo general",
    "3. Objetivos específicos",
    "4. Alcance del proyecto",
    "5. Marco teórico",
    "6. Metodología",
    "7. Requerimientos funcionales",
    "8. Requerimientos no funcionales",
    "9. Diseño de la solución",
    "10. Desarrollo de la implementación",
    "11. Pruebas y validación",
  ].map((x) => p(x, { alignment: AlignmentType.LEFT, after: 45 })),
);

// 1
docChildren.push(
  h1("1. Nombre del proyecto"),
  p([t("Operador de Ingeniería del Caos para Kubernetes", { bold: true, size: 25, color: COLORS.navy })], { alignment: AlignmentType.LEFT, after: 120 }),
  p("Nombre técnico del repositorio: goland-operator. API del proyecto: chaos.engineering.io/v1alpha1."),
  note("Denominación adoptada", "Se conserva el nombre académico del documento original y se usa el API group real del código. No se adopta chaos.operator.io porque no corresponde a la implementación actual."),
);

// 2
docChildren.push(
  h1("2. Objetivo general", false),
  p("Diseñar, implementar y validar un operador de Kubernetes que permita definir y ejecutar de forma declarativa experimentos controlados de fallas sobre pods, red, recursos y comportamiento HTTP, con selección acotada de objetivos, seguimiento del ciclo de vida, recuperación de efectos reversibles y observabilidad mediante mecanismos nativos de Kubernetes y métricas Prometheus."),
  note("Precisión", "El verbo validar expresa una meta del proyecto. La evidencia actual cubre pruebas unitarias y auxiliares, pero no permite afirmar todavía una validación integral en staging o producción.", COLORS.amber),
);

// 3
docChildren.push(
  h1("3. Objetivos específicos", false),
  bullet("Analizar los principios de ingeniería del caos, Kubernetes y el patrón Operator aplicables al problema."),
  bullet("Modelar experimentos mediante cuatro CRD especializados: PodChaos, NetworkChaos, StressChaos y HTTPChaos."),
  bullet("Implementar reconciliadores que gestionen inicialización, ejecución, monitoreo, finalización, fallo y limpieza."),
  bullet("Seleccionar objetivos mediante criterios declarativos y controlar cuantitativamente el blast radius."),
  bullet("Inyectar fallas de pod, red, consumo de recursos y comportamiento HTTP en workloads compatibles."),
  bullet("Registrar estados, condiciones, pods afectados, resultados, eventos, métricas y resúmenes de ejecución."),
  bullet("Aplicar controles de seguridad en el despliegue y completar las validaciones necesarias para impedir abuso de pods/exec y afectación de namespaces protegidos."),
  bullet("Validar la solución con pruebas unitarias, integración, end-to-end, seguridad y carga, dejando evidencia reproducible."),
  bullet("Mantener documentación de instalación, muestras de uso y limitaciones de la versión."),
);

// 4
docChildren.push(
  h1("4. Alcance del proyecto"),
  h2("4.1 Incluido en la versión analizada"),
  bullet("Operador escrito en Go con controller-runtime y cuatro CRD bajo chaos.engineering.io/v1alpha1."),
  bullet("Selección de pods por etiquetas y otros filtros, cinco modos de cantidad y límites maxPods/maxPercentage."),
  bullet("Fallas de pods; reglas de red con tc/iptables; estrés mediante procesos/comandos dentro de contenedores; modificación temporal de Nginx para HTTP."),
  bullet("Fases del experimento, Conditions, timestamps, AffectedPods y resultados por objetivo."),
  bullet("Finalizers y recuperación para efectos de red, estrés y HTTP."),
  bullet("Eventos de Kubernetes, métricas Prometheus, health/readiness y reportes en ConfigMap."),
  bullet("Manifiestos de despliegue, RBAC, samples, entorno kind, monitoring y scripts k6."),
  h2("4.2 Excluido o pendiente"),
  bullet("Ejecución programada real por cron; el campo existe pero no tiene lógica de scheduling."),
  bullet("CRD genéricos ChaosExperiment, ChaosSchedule y ChaosResult descritos en el documento original."),
  bullet("IOChaos y TimeChaos como recursos independientes."),
  bullet("Abortos automáticos por métricas, disponibilidad o tasa de error."),
  bullet("Política configurable de namespaces permitidos/protegidos y exclusión por etiqueta protegida."),
  bullet("Soporte HTTP agnóstico del servidor; la implementación actual está acoplada a Nginx."),
  bullet("Ejecución y certificación en producción, multi-cluster, pruebas de penetración y alta disponibilidad real."),
  h2("4.3 Restricciones técnicas actuales"),
  p("Los experimentos de red, estrés y HTTP ejecutan comandos dentro del contenedor seleccionado. Por tanto, el workload debe contener sh y las herramientas requeridas; NetworkChaos requiere tc/iptables y HTTPChaos requiere Nginx con permisos para modificar su configuración. Estas dependencias reducen la portabilidad de la versión actual."),
  p("No se recomienda promover esta versión a producción hasta cerrar los requisitos de validación de entradas, namespaces protegidos, limpieza selectiva de reglas, semántica HTTP y evidencia end-to-end."),
);

// 5
docChildren.push(
  h1("5. Marco teórico"),
  h2("5.1 Ingeniería del caos"),
  p("La ingeniería del caos es una disciplina experimental orientada a generar confianza en la capacidad de un sistema para soportar condiciones adversas. Un experimento parte de una hipótesis sobre el estado estable, introduce una perturbación controlada, observa métricas relevantes y compara el comportamiento real con el esperado. Su finalidad no es causar fallas por sí mismas, sino descubrir debilidades antes de que se conviertan en incidentes."),
  p("Un principio esencial es reducir el blast radius: comenzar con pocos objetivos, limitar duración y alcance, observar continuamente y disponer de mecanismos de recuperación. Esta idea fundamenta maxPods, maxPercentage, duration y los finalizers del proyecto."),
  h2("5.2 Kubernetes y el patrón Operator"),
  p("Kubernetes administra cargas declarativas mediante una API y controladores que intentan llevar el estado observado hacia el estado deseado. Un Custom Resource Definition amplía la API con un tipo propio; un controller observa sus instancias y ejecuta un reconciliation loop idempotente."),
  p("El patrón Operator codifica conocimiento operativo en ese loop. En este proyecto, cada recurso de caos expresa la falla deseada, los pods objetivo y sus límites. El reconciler selecciona targets, aplica la perturbación, actualiza status y limpia efectos reversibles al finalizar o eliminar el recurso."),
  h2("5.3 Observabilidad y resiliencia"),
  p("Un experimento solo es útil si su impacto puede observarse. Métricas, eventos, logs y estado del recurso permiten responder qué se ejecutó, sobre qué pods, durante cuánto tiempo y con qué resultado. Las métricas de latencia, errores, disponibilidad, throughput y saturación pertenecen al sistema bajo prueba; las métricas del operador describen su propia reconciliación y ejecución."),
  h2("5.4 Seguridad en experimentos de caos"),
  p("Un operador de caos concentra capacidades sensibles: eliminación de pods y ejecución de comandos en contenedores. La seguridad exige mínimo privilegio, validación estricta de los Custom Resources, exclusiones de namespaces críticos, límites conservadores, recuperación determinista y trazabilidad. Los valores externos nunca deben convertirse directamente en fragmentos de shell o configuración ejecutable."),
  h2("5.5 Referencias base"),
  p("Basiri et al. (2016), “Chaos Engineering”, IEEE Software, 33(3), 35–41. DOI: 10.1109/MS.2016.60."),
  p("Rosenthal y Jones (2020), Chaos Engineering: System Resiliency in Practice. O’Reilly Media."),
  p("Burns, Beda y Hightower (2019), Kubernetes: Up and Running, 2.ª ed. O’Reilly Media."),
  p("Dobies y Wood (2020), Kubernetes Operators: Automating the Container Orchestration Platform. O’Reilly Media."),
  p("Principles of Chaos Engineering (2019), principlesofchaos.org; Kubernetes Documentation (2024), Operator Pattern y Kubernetes Components."),
);

// 6
docChildren.push(
  h1("6. Metodología"),
  h2("6.1 Enfoque de desarrollo reconstruido"),
  p("La evidencia del repositorio muestra un desarrollo incremental y orientado a componentes. Se definieron primero tipos de API, interfaces de dominio y operaciones de infraestructura; posteriormente se agregaron reconcilers, validaciones, observabilidad, muestras y pruebas. No se encontró evidencia suficiente para afirmar que se siguió formalmente Scrum, Kanban, Domain-Driven Design completo o un flujo GitOps; esos términos no deben presentarse como metodología ejecutada sin artefactos adicionales."),
  h2("6.2 Ciclo de trabajo propuesto para completar el proyecto"),
  step("Definir una hipótesis verificable y métricas de estado estable para cada experimento."),
  step("Especificar el CR, selector, duración, blast radius, criterios de aborto y recuperación."),
  step("Implementar o ajustar API, webhook, reconciler y operación de infraestructura."),
  step("Ejecutar format, lint, pruebas unitarias y análisis de seguridad; corregir antes de desplegar."),
  step("Validar en kind con un workload controlado y recopilar evidencia antes/durante/después."),
  step("Ejecutar pruebas de carga y revisar métricas del operador."),
  step("Documentar resultado, limitaciones, decisión de aceptación y trabajo pendiente."),
  h2("6.3 Criterio de trazabilidad"),
  p("Cada requerimiento y caso de prueba debe conservar un identificador estable. La evidencia mínima debe indicar versión del código, entorno, comando, resultado observado, fecha y responsable. Una funcionalidad descrita pero no cubierta por código o prueba se mantiene como pendiente."),
);

// 7
docChildren.push(
  h1("7. Requerimientos funcionales"),
  p("La siguiente matriz expresa el comportamiento esperado y su estado frente al código revisado. “Parcial” no equivale a defecto cerrado: identifica una capacidad que necesita definición, implementación o validación adicional."),
  table(["ID", "Requerimiento", "Estado", "Evidencia / observación"], functionalRows, [760, 3830, 1180, 3868], { bodySize: 17 }),
);

// 8
docChildren.push(
  h1("8. Requerimientos no funcionales"),
  p("Los atributos no funcionales se formulan de manera verificable cuando existe evidencia. En los demás casos se evita fijar cifras arbitrarias: el umbral debe aprobarse antes de la prueba."),
  table(["ID", "Atributo", "Requerimiento", "Estado", "Base"], nonFunctionalRows, [700, 1200, 3530, 1320, 2888], { bodySize: 16 }),
);

// 9
docChildren.push(
  h1("9. Diseño de la solución"),
  h2("9.1 Vista de componentes"),
  table(
    ["Componente", "Responsabilidad", "Interacción principal"],
    [
      ["Kubernetes API Server", "Persiste CRD, Pods, Events, ConfigMaps y status.", "Recibe YAML/kubectl y notifica cambios a los controllers."],
      ["Manager controller-runtime", "Administra scheme, caché, leader election, métricas, probes y webhooks opcionales.", "Registra los cuatro reconcilers."],
      ["Reconcilers", "Gestionan ciclo de vida, finalizers, selección, ejecución, status, eventos y reportes.", "Delegan selección al dominio y ejecución a infraestructura."],
      ["TargetSelector", "Lista candidatos, aplica el modo y limita el blast radius.", "Consume PodLister y devuelve pods objetivo."],
      ["Operaciones de infraestructura", "Eliminan pods o ejecutan comandos dentro de contenedores.", "Usan Kubernetes client y pods/exec."],
      ["Observabilidad", "Publica métricas y conserva resultados resumidos.", "Prometheus, Events, status y ConfigMaps."],
      ["Workloads objetivo", "Aplicaciones de prueba sobre las cuales se introduce la falla.", "Deben incluir herramientas/permisos compatibles con el tipo de caos."],
    ],
    [2100, 3700, 3438],
    { bodySize: 17 },
  ),
  h2("9.2 Flujo de reconciliación"),
  step("El usuario crea un PodChaos, NetworkChaos, StressChaos o HTTPChaos.", "stepsDesign"),
  step("El API Server aplica el esquema CRD y, si están activos, los webhooks agregan defaults y validan.", "stepsDesign"),
  step("El reconciler agrega un finalizer, registra StartTime y lleva el recurso a Pending.", "stepsDesign"),
  step("TargetSelector obtiene candidatos, aplica el modo y el blast radius.", "stepsDesign"),
  step("La operación correspondiente inyecta la falla y el reconciler registra AffectedPods, Result, métricas y Events.", "stepsDesign"),
  step("El recurso permanece Running hasta cumplir duration; PodChaos puede ejecutar ondas mediante una anotación.", "stepsDesign"),
  step("Los efectos reversibles se limpian al finalizar o borrar el CR; luego se registra Completed/Failed y se crea el reporte.", "stepsDesign"),
  h2("9.3 Modelo de recursos"),
  table(
    ["CRD", "Finalidad", "Acciones / parámetros principales"],
    [
      ["PodChaos", "Simular pérdida o reinicio de pods/contenedores.", "pod-kill, pod-failure, container-kill; gracePeriod, containerNames."],
      ["NetworkChaos", "Alterar comunicación del pod.", "delay, loss, duplicate, corrupt, partition, bandwidth; tc/iptables."],
      ["StressChaos", "Aumentar consumo de recursos.", "CPU, memoria, disk-fill e I/O; workers, size, path."],
      ["HTTPChaos", "Modificar temporalmente respuestas/configuración HTTP.", "abort, delay, replace, patch; target, port, path, method."],
    ],
    [1800, 3300, 4138],
    { bodySize: 17 },
  ),
  h2("9.4 Decisiones de seguridad pendientes"),
  bullet("Aplicar validaciones allowlist a todos los valores usados en sh -c o en configuración Nginx."),
  bullet("Definir namespaces permitidos y protegidos y una etiqueta de exclusión obligatoria."),
  bullet("Reducir RBAC a los verbos realmente necesarios y separar permisos por modo de despliegue si corresponde."),
  bullet("Reemplazar math/rand y acordar si la selección debe ser reproducible para pruebas o no predecible."),
  bullet("Hacer que la recuperación de red elimine únicamente las reglas creadas por el experimento."),
  bullet("Definir y probar criterios de aborto automático antes de cualquier uso en producción."),
);

// 10
docChildren.push(
  h1("10. Desarrollo de la implementación", false),
  h2("10.1 Stack tecnológico comprobado"),
  table(
    ["Tecnología", "Versión / uso comprobado"],
    [
      ["Go", "1.22.5 (go.mod)"],
      ["controller-runtime", "v0.17.0"],
      ["Kubernetes libraries", "k8s.io/api, apimachinery y client-go v0.29.0"],
      ["Prometheus client", "github.com/prometheus/client_golang v1.18.0"],
      ["Contenedores y clúster local", "Docker + kind; versiones mínimas documentadas, no revalidadas en esta revisión"],
      ["Pruebas de carga", "k6 + xk6-kubernetes, con escenarios smoke/steady/spike"],
    ],
    [2900, 6338],
    { bodySize: 18 },
  ),
  h2("10.2 Implementación por módulo"),
  table(["Módulo", "Ruta", "Responsabilidad", "Estado"], implementationRows, [1750, 2100, 3870, 1518], { bodySize: 16 }),
  h2("10.3 Diferencias entre el documento original y el código"),
  table(["Tema", "Documento original", "Código actual", "Tratamiento recomendado"], differenceRows, [1450, 2580, 2580, 2628], { bodySize: 16 }),
  h2("10.4 Limitaciones que deben quedar expresas"),
  bullet("HTTPChaos delay no produce una demora real en la versión actual; devuelve una respuesta con headers indicativos."),
  bullet("HTTPChaos patch no tiene script de ejecución y varios filtros declarados —porcentaje, método y dirección request/response— no se aplican completamente."),
  bullet("NetworkChaos partition afecta INPUT y OUTPUT completos del contenedor y no usa el target declarado para limitar pares de comunicación."),
  bullet("StressChaos CPU no respeta el porcentaje load; algunas opciones del API todavía no se traducen a la ejecución."),
  bullet("SchedulerSpec, Force y varios selectores/parámetros están modelados pero no tienen comportamiento completo."),
  bullet("Los scripts construidos con datos del CR requieren validación estricta antes de considerar la solución segura."),
);

// 11
docChildren.push(
  h1("11. Pruebas y validación"),
  h2("11.1 Estado verificado durante este análisis"),
  p("El 20 de agosto de 2026 se ejecutó go test ./... con resultado satisfactorio en los paquetes con pruebas. La cobertura global obtenida fue 13,3 % de statements: api/v1alpha1 11,1 %, controllers 12,7 %, internal/app 3,6 %, internal/domain 52,3 % e internal/infrastructure 0,0 %. Esto confirma que la suite actual pasa, pero no demuestra validación integral."),
  p("No se ejecutaron pruebas end-to-end, de seguridad ni de carga en un clúster durante esta revisión. El repositorio contiene artefactos para kind, monitoring y k6; sus resultados deben adjuntarse por separado."),
  h2("11.2 Casos de prueba y evidencia a completar"),
  table(["ID", "Nivel", "Acción", "Resultado esperado", "Evidencia a adjuntar"], testRows, [650, 1050, 2970, 3250, 1318], { bodySize: 15 }),
  h2("11.3 Ficha de evidencia"),
  table(
    ["Campo", "Contenido a completar"],
    [
      ["Caso / requerimiento", "CP-__ / RF-__ / RNF-__"],
      ["Versión", "Commit, tag o hash de imagen. El directorio analizado no expuso metadata Git."],
      ["Entorno", "Clúster, versión Kubernetes, nodos, runtime y namespace."],
      ["Fecha y responsable", "AAAA-MM-DD, nombre y rol."],
      ["Precondiciones", "Workload, réplicas, métricas de estado estable y permisos."],
      ["Comandos", "Comandos exactos o referencia al script utilizado."],
      ["Resultado observado", "Datos medidos; evitar únicamente “funciona”."],
      ["Evidencias", "Capturas, logs, kubectl describe, Events, PromQL, reporte k6 y archivos."],
      ["Conclusión", "Pasó / Falló / Bloqueado, defecto asociado y decisión."],
    ],
    [2300, 6938],
    { bodySize: 17 },
  ),
  h2("11.4 Criterios de cierre por acordar"),
  bullet("Cero defectos críticos o altos en seguridad, cleanup y blast radius."),
  bullet("Todos los RF declarados para la entrega con al menos un caso ejecutado y evidencia."),
  bullet("Umbral de cobertura automatizada aprobado; el valor no se fija aquí porque la cobertura actual es 13,3 %."),
  bullet("SLO de rendimiento y carga aprobado y cumplido; loadtest/ documenta valores iniciales que deben ser ratificados."),
  bullet("Pruebas de recuperación exitosas ante duration, borrado del CR y reinicio del operador."),
  bullet("Aprobación explícita del entorno permitido: desarrollo, staging o producción."),
  spacer(200),
  note("Cierre", "El proyecto cuenta con una base funcional y documental significativa. El siguiente paso formal no es ampliar el alcance, sino decidir las capacidades pendientes, cerrar las brechas de seguridad y semántica, y reunir evidencia reproducible para cada requisito aceptado.", COLORS.green),
);

const doc = new Document({
  creator: "Codex",
  title: "Operador de Ingeniería del Caos para Kubernetes — Documento técnico",
  subject: "Formulación, requerimientos, diseño, implementación y plan de validación",
  description: "Documento nuevo elaborado a partir del archivo académico y el código fuente del proyecto goland-operator.",
  numbering,
  styles: {
    default: {
      document: {
        run: { font: "Aptos", size: 21, color: "20242A" },
        paragraph: { spacing: { line: 276, after: 100 } },
      },
    },
    paragraphStyles: [
      {
        id: "Title",
        name: "Title",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { font: "Aptos Display", size: 32, bold: true, color: COLORS.navy },
        paragraph: { spacing: { after: 200 }, outlineLevel: 0 },
      },
      {
        id: "Heading1",
        name: "Heading 1",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { font: "Aptos Display", size: 30, bold: true, color: COLORS.navy },
        paragraph: { spacing: { before: 0, after: 180 }, outlineLevel: 0 },
      },
      {
        id: "Heading2",
        name: "Heading 2",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { font: "Aptos Display", size: 24, bold: true, color: COLORS.blue },
        paragraph: { spacing: { before: 220, after: 100 }, outlineLevel: 1 },
      },
      {
        id: "Heading3",
        name: "Heading 3",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { font: "Aptos", size: 21, bold: true, color: COLORS.gray },
        paragraph: { spacing: { before: 150, after: 70 }, outlineLevel: 2 },
      },
    ],
  },
  sections: [
    {
      properties: {
        page: {
          size: { width: PAGE_WIDTH, height: PAGE_HEIGHT },
          margin: { top: 980, bottom: 980, left: MARGIN, right: MARGIN, header: 500, footer: 500 },
        },
      },
      headers: {
        default: new Header({
          children: [
            new Paragraph({
              children: [t("OPERADOR DE INGENIERÍA DEL CAOS PARA KUBERNETES", { bold: true, size: 16, color: COLORS.gray })],
              alignment: AlignmentType.RIGHT,
              border: { bottom: { style: BorderStyle.SINGLE, size: 4, color: COLORS.border, space: 4 } },
              spacing: { after: 60 },
            }),
          ],
        }),
      },
      footers: {
        default: new Footer({
          children: [
            new Paragraph({
              children: [
                t("Documento técnico — Versión 1.0", { size: 16, color: COLORS.gray }),
                t("     |     ", { size: 16, color: COLORS.gray }),
                t("Página ", { size: 16, color: COLORS.gray }),
                new TextRun({ children: [PageNumber.CURRENT], size: 16, color: COLORS.gray }),
              ],
              alignment: AlignmentType.CENTER,
              spacing: { before: 60 },
            }),
          ],
        }),
      },
      children: docChildren,
    },
  ],
});

Packer.toBuffer(doc).then((buffer) => {
  fs.writeFileSync(OUTPUT, buffer);
  process.stdout.write(`${OUTPUT}\n`);
});
