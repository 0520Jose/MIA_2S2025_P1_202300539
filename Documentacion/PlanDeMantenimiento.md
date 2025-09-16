# PLAN DE MANTENIMIENTO
## Sistema de Administración de Discos Virtuales ExtreamFS

### INFORMACIÓN DEL PROYECTO
- **Nombre del Proyecto:** ExtreamFS
- **Desarrollado por:** José Emanuel Monzón Lémus
- **Institución/Organización:** Universidad de San Carlos de Guatemala
- **Fecha del Plan:** 15/09/2025
- **Versión Actual:** 1.0

---

## RESUMEN EJECUTIVO

Este documento establece un plan de mantenimiento y desarrollo continuo para ExtreamFS, una aplicación educativa para simular sistemas de archivos EXT2. El plan está diseñado para asegurar la estabilidad, relevancia y evolución del sistema a lo largo del tiempo, considerando su naturaleza como proyecto universitario con potencial para expandirse dentro del ámbito académico. Se definen los procesos, roles, cronogramas y estrategias para garantizar que el sistema siga siendo una herramienta valiosa para la enseñanza de sistemas operativos.

---

## ESTRATEGIAS DE MANTENIMIENTO

### 1. Mantenimiento Correctivo

- **Gestión de Errores**:
  - Implementar un sistema de reporte de errores a través de GitHub Issues
  - Establecer prioridades basadas en la severidad e impacto educativo
  - Resolución trimestral de errores críticos reportados

- **Verificación de Compatibilidad**:
  - Actualización semestral de compatibilidad con navegadores web comunes
  - Revisión de dependencias y librerías para prevenir vulnerabilidades

- **Documentación de Cambios**:
  - Mantener un registro de cambios (CHANGELOG.md) en el repositorio
  - Actualizar la documentación técnica cuando se realicen correcciones importantes

### 2. Mantenimiento Adaptativo

- **Adaptación Tecnológica**:
  - Actualización anual del backend Go y dependencias frontend
  - Monitoreo de cambios en especificaciones EXT2 y ajustes correspondientes
  - Optimizaciones para compatibilidad con nuevos sistemas operativos

- **Adaptación Pedagógica**:
  - Revisión semestral de requisitos educativos con docentes de sistemas operativos
  - Ajustes en función de retroalimentación de estudiantes y resultados de aprendizaje

### 3. Mantenimiento Preventivo

- **Pruebas Periódicas**:
  - Ejecución trimestral de pruebas automatizadas para comandos core
  - Verificación de generación de reportes y validación estructural
  - Pruebas de carga para garantizar rendimiento con grupos grandes de estudiantes

- **Optimización de Código**:
  - Revisión semestral de componentes críticos del sistema
  - Refactorización de secciones identificadas con problemas potenciales
  - Implementación de mejores prácticas conforme evolucionen los lenguajes utilizados

---

## CICLO DE ACTUALIZACIONES

### Actualizaciones Menores (cada 3-4 meses)
- Corrección de errores reportados
- Mejoras en interfaz de usuario
- Optimizaciones de rendimiento
- Actualización de documentación

### Actualizaciones Principales (anual)
- Implementación de nuevas funcionalidades
- Renovación tecnológica de componentes obsoletos
- Expansiones importantes de características educativas
- Actualizaciones de arquitectura si se requieren

---

## PLAN DE DESARROLLO FUTURO

### Fase 1: Consolidación (6-12 meses)
1. **Mejora de Interfaz de Usuario**
   - Implementar tema oscuro y opciones de accesibilidad
   - Añadir más elementos interactivos para manipulación de archivos
   - Mejorar experiencia móvil para acceso desde cualquier dispositivo

2. **Optimización de Reportes**
   - Añadir opciones de exportación (PDF, PNG) para reportes generados
   - Mejorar visualización de estructuras complejas
   - Implementar zoom y navegación avanzada en reportes gráficos

3. **Expansión de Documentación**
   - Desarrollo de tutoriales interactivos paso a paso
   - Creación de ejercicios prácticos para laboratorios
   - Traducción de documentación a otros idiomas (inglés)

### Fase 2: Expansión (12-24 meses)
1. **Nuevos Sistemas de Archivos**
   - Implementar soporte para EXT3/EXT4
   - Añadir FAT32 para comparación de diferentes sistemas
   - Desarrollar visualizaciones comparativas entre sistemas

2. **Funcionalidades Colaborativas**
   - Sistema básico de usuarios para trabajo en equipo
   - Compartición de discos virtuales entre estudiantes
   - Módulo para profesores para revisar trabajo de estudiantes

3. **Integraciones Académicas**
   - API para integración con plataformas LMS (Moodle)
   - Sistema de evaluación automática de ejercicios
   - Generación de informes de progreso para docentes

### Fase 3: Innovación (2+ años)
1. **Simulación de Escenarios Avanzados**
   - Fallos de sistema y recuperación
   - Simulación de ataques y seguridad de archivos
   - Optimización y fragmentación de discos

2. **Extensión a Otros Conceptos de SO**
   - Gestión básica de memoria
   - Simulación de procesos
   - Programación concurrente básica

3. **Plataforma de Código Abierto**
   - Establecer comunidad de desarrollo
   - Implementar sistema de plugins/extensiones
   - Crear documentación para contribuidores externos

---

## RECURSOS Y RESPONSABILIDADES

### Mantenimiento Regular
- **Desarrollador Principal**: Estudiante responsable del proyecto (5-10 horas mensuales)
- **Soporte Académico**: Docente asesor para verificar relevancia educativa (2-4 horas mensuales)
- **Pruebas**: Estudiantes voluntarios de cursos de sistemas operativos (según disponibilidad)

### Desarrollo de Nuevas Funcionalidades
- **Equipo de Desarrollo**: Posibles estudiantes de EPS o proyectos derivados
- **Asesores de Contenido**: Docentes de cursos relacionados con sistemas operativos
- **Colaboradores Externos**: Comunidad universitaria interesada en el proyecto

---

## MÉTRICAS DE MANTENIMIENTO

Para evaluar la efectividad del plan de mantenimiento, se monitorizarán:

1. **Métricas Técnicas**:
   - Número de errores reportados/resueltos por trimestre
   - Tiempo medio de resolución de problemas críticos
   - Cobertura de pruebas automatizadas

2. **Métricas de Uso**:
   - Número de estudiantes utilizando el sistema
   - Frecuencia de uso de diferentes características
   - Tiempo promedio de sesión

3. **Métricas Educativas**:
   - Mejora en comprensión de conceptos (mediante encuestas)
   - Satisfacción de docentes con la herramienta
   - Impacto en resultados académicos

---

## CONSIDERACIONES PRESUPUESTARIAS

Como proyecto universitario, ExtreamFS debe mantener un enfoque de bajo costo. Se sugieren las siguientes estrategias:

- **Infraestructura**: Utilizar servicios gratuitos para hosting (GitHub Pages, Netlify, etc.)
- **Desarrollo**: Incorporar el mantenimiento como proyectos para estudiantes avanzados
- **Herramientas**: Priorizar software de código abierto para todas las necesidades de desarrollo
- **Documentación**: Utilizar wikis y plataformas colaborativas gratuitas

Para desarrollos más ambiciosos, se podrían buscar:
- Fondos de innovación educativa de la universidad
- Colaboraciones con otros cursos relacionados 
- Patrocinios pequeños de departamentos académicos interesados

---