# Manual De Usuario
**UNIVERSIDAD DE SAN CARLOS DE GUATEMALA**  
**FACULTAD DE INGENIERÍA**  
**PRIMER PROYECTO - MANEJO E IMPLEMENTACIÓN DE ARCHIVOS**  
**CATEDRÁTICO: ING. WILLIAM ESCOBAR**  
**TUTOR ACADÉMICO: DANIEL MELLADO**

**NOMBRE DEL ESTUDIANTE: ** Javier Andrés Velásquez Bonilla 
**CARNÉ:** 202307775 
**SECCIÓN:** A-  

**GUATEMALA, [11/09/2025]**



# **OBJETIVOS DEL SISTEMA** 

## **GENERAL** 

El objetivo general de este manual es proporcionar una guía completa y detallada para el usuario, facilitando la comprensión y el uso efectivo del programa desarrollado para analizar la saturación de mercados en diferentes países.

## **ESPECÍFICOS**

1. Facilitar al usuario la comprensión de la interfaz gráfica de la aplicación, incluyendo la navegación entre las diferentes secciones, el uso de los botones y opciones disponibles, y cómo acceder a la documentación de ayuda para resolver dudas sobre su funcionamiento.
2. Detallar el manejo de errores y cómo el usuario puede obtener reportes de análisis léxico cuando se encuentran problemas en los datos proporcionados.

# **INTRODUCCIÓN** 

Este manual está diseñado para guiar al usuario en el uso adecuado de la aplicación desarrollada para analizar la saturación de mercado en diferentes países, utilizando un archivo en formato .ORG. La aplicación permite identificar el país más adecuado para establecer un negocio y visualiza la información mediante gráficos generados con Graphviz. La interfaz gráfica de usuario, desarrollada con Tkinter en Python, permite cargar archivos de entrada, procesar datos y presentar recomendaciones de forma visual.

Este manual cubre los siguientes temas:

* Cómo cargar y analizar archivos .ORG  
* Uso del editor de texto integrado para la creación y modificación de datos  
* Cómo interpretar los resultados y las recomendaciones proporcionadas por la aplicación  
* Manejo de errores en el análisis léxico y generación de reportes

# **INFORMACIÓN DEL SISTEMA**

El programa funciona mediante la lectura y análisis de archivos en formato .ORG que contienen información sobre los países y sus niveles de saturación de mercado. El flujo básico de operaciones es el siguiente:

1. **Carga del Archivo de Entrada**: El programa permite cargar un archivo .ORG que contiene las propuestas de países, así como datos sobre su población y saturación de mercado.  
2. **Análisis de Datos**: El sistema analiza el archivo utilizando un analizador léxico en Fortran para procesar los datos e identificar posibles errores en el formato de las instrucciones. En caso de errores, se genera un informe de errores en formato HTML.  
3. **Generación de Gráfico**: Tras un análisis exitoso, el programa genera un gráfico que muestra los continentes y países junto con sus niveles de saturación. Además, selecciona el país con la menor saturación como la mejor opción y lo muestra en la interfaz junto con su bandera y población.  
4. **Reporte y Visualización**: El gráfico generado se muestra en la interfaz gráfica, junto con el reporte de análisis léxico, lo que facilita al usuario visualizar los datos y tomar decisiones basadas en la información procesada.

# **REQUISITOS DEL SISTEMA**

Para ejecutar este programa, el usuario necesita lo siguiente:

* **Sistema Operativo:** Windows, macOS, o Linux.  
* **Compilador Fortran:** Un compilador compatible con el estándar Fortran 90 o superior.  
* **Editor de Texto/IDE:** Cualquier editor de texto que soporte Fortran, como VS Code, Atom, o cualquier otro IDE especializado en Fortran.  
* **Archivos de Entrada:** Se requieren archivos en formato .ORG que contengan las propuestas de países y datos de saturación.

# **FLUJO DE LAS FUNCIONALIDADES DEL SISTEMA** 


Tenemos el siguiente proyecto con el propósito de permitir al cliente analizar la saturación del mercado, con el fin de poner una oficina, tomando en cuenta diferentes países, utilizando un archivo con diversa información, con extensión .ORG. La aplicación combina un análisis léxico realizado en Fortran con una interfaz gráfica desarrollada en Python, a través de la biblioteca Tkinter.

Para la utilización del mismo, se requiere que se pueda descargar el repositorio completo. Para esto se descarga el .zip del release propuesto, por lo que debemos descomprimirlo. Luego abrir la terminal en la carpeta del proyecto y luego introducir `python fortran_gui.py`. Esto, básicamente sirve para acceder al archivo .py en donde se encuentra el main de Python y permite ejecutarlo. Dicho esto, se le desplegará la aplicación, la cual luce así:

![Home](./Imagenes/Interfaz.JPG)

Dentro del mismo, usted cuenta con un epsacio para editar, escribir texto o bien, existe la posibilidad de subir un archivo .ORG para que sea analizado posteriormente. En la sección de menú, usted cuenta con 3 opciones, las cuales son: Guardar, Guardar como y Abrir, además de un apartado de "Ayuda" en el cual encuentra información acerca del desarrollador. Luego de poner en el input lo que desea que se analice, se verá más o menos así:

![Editor de texto lleno](./Imagenes/Editor.JPG)

Cuando haya cargado todo lo que desea analizar, simplemente presione el botón que está abajo del editor de texto y espere a los resultados. Cuando ya se haya calculado todo, se mostrarán en pantalla los datos obtenidos por el analizador.

![Resultados del análisis](./Imagenes/Analizado.JPG)

Si está todo correcto con el archivo o texto de entrada, adiocionalmente se generará un HTML indicando todos los lexemas y tokens encontrados.

![Tabla de Lexemas](./Imagenes/tokens.JPG)

De igual forma, si al momento de usar el editor de texto o cargar un archivo .ORG. se presenta algún error léxico, el programa no le devolverá el resultado, sino que solamente generará un HTML con el listado de errores encontrados en el área de texto o la entrada ingresada en el formato antes mencionado. 

![Error en el editor](./Imagenes/errores.JPG)

El programa tiene la capacidad de guardar las modificaciones que se realicen en el editor de texto, tanto si conservan el mismo nombre como si se quiere modificar esto, sin embargo, no el resultado de los análisis.
