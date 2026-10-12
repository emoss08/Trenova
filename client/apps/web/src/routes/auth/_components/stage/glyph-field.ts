import { hexToSrgb, type Srgb } from "@/lib/oklch";

const VERTEX_SHADER = `#version 300 es
in vec2 p;void main(){gl_Position=vec4(p,0.,1.);}`;

const FRAGMENT_SHADER = `#version 300 es
precision highp float;
uniform vec2 uRes;uniform float uTime;uniform vec2 uMouse;uniform float uHover;
uniform float uPulse;uniform float uDone;uniform float uIntro;uniform vec2 uCell;
uniform sampler2D uAtlas;uniform float uGlyphs;uniform vec3 uBg,uC0,uC1,uC2,uC3;uniform float uDots;
out vec4 o;
float hash(vec2 p){return fract(sin(dot(p,vec2(127.1,311.7)))*43758.5453);}
float noise(vec2 p){vec2 i=floor(p),f=fract(p);vec2 u=f*f*(3.-2.*f);
 return mix(mix(hash(i),hash(i+vec2(1,0)),u.x),mix(hash(i+vec2(0,1)),hash(i+vec2(1,1)),u.x),u.y);}
float fbm(vec2 p){float v=0.,a=.5;for(int i=0;i<4;i++){v+=a*noise(p);p*=2.03;a*=.5;}return v;}
vec3 pal(float x){x=clamp(x,0.,1.);
 return x<.333?mix(uC0,uC1,x*3.):x<.666?mix(uC1,uC2,(x-.333)*3.):mix(uC2,uC3,(x-.666)*3.);}
vec3 field(vec2 px,float t){
 float asp=uRes.x/uRes.y;
 vec2 p=px/uRes;p.x*=asp;
 vec2 m=uMouse/uRes;m.x*=asp;
 float md=length(p-m);
 float a=-.62;mat2 R=mat2(cos(a),-sin(a),sin(a),cos(a));
 vec2 q=R*(p-vec2(.58*asp,.5));
 float w=fbm(q*1.5+vec2(0.,t*.06));
 float c=.13*sin(q.y*2.1+t*.22)+.05*sin(q.y*5.3-t*.37)+(w-.5)*.22;
 c+=uHover*.06*exp(-md*md*28.)*sin(md*30.-t*3.);
 float twist=abs(sin(q.y*1.25+t*.13+.6));
 float width=(.035+.2*twist)*(1.+uDone*.35);
 float d=(q.x-c)/width;
 float body=1.-smoothstep(.45,1.,abs(d));
 float fib=.5+.5*sin(d*(9.+8.*(1.-twist))+fbm(q*3.5+t*.08)*5.);
 float fold=(1.-twist)*.55;
 float I=body*(.32+.5*fib+fold);
 I+=uHover*.35*exp(-md*md*90.)*(.5+.5*sin(md*60.-t*4.));
 return vec3(I,d,q.y);
}
void main(){
 vec2 cell=floor(gl_FragCoord.xy/uCell);
 vec2 ctr=(cell+.5)*uCell;
 vec2 loc=fract(gl_FragCoord.xy/uCell);
 float t=uTime;
 vec3 f=field(ctr,t);
 float I=f.x;
 float sx=ctr.x/uRes.x;
 float reveal=smoothstep(sx-.15,sx+.05,uIntro*1.25-.1);
 I*=reveal;
 float dx=ctr.x/uRes.x;
 float front=uPulse*1.5-.2;
 float wave=exp(-pow((dx-front)*7.,2.))*(1.-uPulse);
 I+=wave*(.55+.45*hash(cell+floor(t*12.)));
 float n=hash(cell);
 float dots=step(.5,fract((cell.x+cell.y)*.5))*uDots*(.55+.45*sin(t*.6+n*6.28));
 float sparkle=step(.996,hash(cell+floor(t*.7)))*.35;
 float g=floor(clamp(I,0.,.999)*uGlyphs);
 float alpha=0.;vec3 col=pal((f.y+1.)*.5+.12*sin(f.z*2.+t*.1));
 if(I>.06){
  alpha=texture(uAtlas,vec2((g+loc.x)/uGlyphs,1.-loc.y)).a*clamp(.35+I,0.,1.);
 } else {
  alpha=texture(uAtlas,vec2((1.+loc.x)/uGlyphs,1.-loc.y)).a*dots*.28;col=mix(uC0,uBg,.15);
  if(sparkle>0.){alpha=texture(uAtlas,vec2((3.+loc.x)/uGlyphs,1.-loc.y)).a*sparkle;}
 }
 vec3 fp=field(gl_FragCoord.xy,t);
 float wash=(1.-smoothstep(.2,1.,abs(fp.y)))*.07*smoothstep(sx-.15,sx+.05,uIntro*1.25-.1);
 vec3 base=mix(uBg,pal((fp.y+1.)*.5),wash);
 o=vec4(mix(base,col,alpha),1.);
}`;

const RAMP = " .:-=+*#%@";
const RAMP_GLYPHS = RAMP.split("");
const MAX_DPR = 2;
const TIME_OFFSET = 12;
const INTRO_SECONDS = 1.8;
const BURST_SECONDS = 1.3;
const MAX_FRAME_SECONDS = 0.05;
const CELL_WIDTH_RATIO = 0.62;
const GLYPH_SIZE_RATIO = 0.86;
const OFFSCREEN = -999;

const UNIFORMS = [
  "uRes",
  "uTime",
  "uMouse",
  "uHover",
  "uPulse",
  "uDone",
  "uIntro",
  "uCell",
  "uAtlas",
  "uGlyphs",
  "uBg",
  "uC0",
  "uC1",
  "uC2",
  "uC3",
  "uDots",
] as const;

type UniformName = (typeof UNIFORMS)[number];

export type GlyphPalette = readonly [string, string, string, string];

export type GlyphFieldOptions = {
  /** Glyph cell height in CSS pixels. */
  cell: number;
  speed: number;
  /** Four `#rrggbb` stops across the ribbon, outer to inner. */
  palette: GlyphPalette;
  /** The ground under the glyphs, `#rrggbb`. */
  background: string;
  /** 0–1 weight of the breathing dot grid around the ribbon. */
  dots: number;
  /** False renders one still frame per change and runs no loop. */
  motion: boolean;
  /** The CSS font stack the glyph atlas is drawn in. */
  fontFamily: string;
};

type Program = {
  program: WebGLProgram;
  buffer: WebGLBuffer;
  texture: WebGLTexture;
  uniforms: Record<UniformName, WebGLUniformLocation | null>;
};

const BLACK: Srgb = [0, 0, 0];

function toSrgb(hex: string): Srgb {
  return hexToSrgb(hex) ?? BLACK;
}

function compile(gl: WebGL2RenderingContext, type: number, source: string): WebGLShader | null {
  const shader = gl.createShader(type);
  if (!shader) {
    return null;
  }
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    gl.deleteShader(shader);
    return null;
  }
  return shader;
}

function buildProgram(gl: WebGL2RenderingContext): Program | null {
  const vertex = compile(gl, gl.VERTEX_SHADER, VERTEX_SHADER);
  const fragment = compile(gl, gl.FRAGMENT_SHADER, FRAGMENT_SHADER);
  const program = gl.createProgram();
  const buffer = gl.createBuffer();
  const texture = gl.createTexture();
  if (!vertex || !fragment || !program || !buffer || !texture) {
    return null;
  }

  gl.attachShader(program, vertex);
  gl.attachShader(program, fragment);
  gl.linkProgram(program);
  gl.deleteShader(vertex);
  gl.deleteShader(fragment);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    gl.deleteProgram(program);
    return null;
  }
  gl.useProgram(program);

  gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
  gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
  const position = gl.getAttribLocation(program, "p");
  gl.enableVertexAttribArray(position);
  gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0);

  const uniforms = {} as Record<UniformName, WebGLUniformLocation | null>;
  for (const name of UNIFORMS) {
    uniforms[name] = gl.getUniformLocation(program, name);
  }

  return { program, buffer, texture, uniforms };
}

/**
 * A twisting ribbon drawn as a grid of monospace glyphs. It renders and nothing else:
 * the owner decides when it exists, when it runs, and what palette it shows.
 *
 * Under `motion: false` there is no animation loop at all — each change draws one
 * still frame — and the palette snaps instead of easing.
 */
export class GlyphField {
  private readonly canvas: HTMLCanvasElement;
  private readonly gl: WebGL2RenderingContext;
  private readonly atlas: HTMLCanvasElement;
  private readonly resizeObserver: ResizeObserver;
  private program: Program | null;
  private options: GlyphFieldOptions;

  private dpr = 1;
  private cellWidth = 1;
  private cellHeight = 1;
  private time = 0;
  private last = 0;
  private frame = 0;
  private running = false;
  private intro = 0;
  private pulse = 1;
  private done = 0;
  private doneTarget = 0;
  private hover = 0;
  private hoverTarget = 0;
  private mouse: [number, number] = [OFFSCREEN, OFFSCREEN];
  private smoothedMouse: [number, number] = [OFFSCREEN, OFFSCREEN];
  private palette: [number, number, number][];
  private paletteTarget: Srgb[];
  private background: Srgb;

  /** Null when the browser has no WebGL2 or the shader does not build. */
  static create(canvas: HTMLCanvasElement, options: GlyphFieldOptions): GlyphField | null {
    if (typeof WebGL2RenderingContext === "undefined") {
      return null;
    }
    const gl = canvas.getContext("webgl2", {
      antialias: false,
      premultipliedAlpha: false,
      preserveDrawingBuffer: true,
    });
    if (!gl) {
      return null;
    }
    const program = buildProgram(gl);
    if (!program) {
      return null;
    }
    return new GlyphField(canvas, gl, program, options);
  }

  private constructor(
    canvas: HTMLCanvasElement,
    gl: WebGL2RenderingContext,
    program: Program,
    options: GlyphFieldOptions,
  ) {
    this.canvas = canvas;
    this.gl = gl;
    this.program = program;
    this.options = options;
    this.atlas = document.createElement("canvas");
    this.paletteTarget = options.palette.map(toSrgb);
    this.palette = this.paletteTarget.map((stop) => [...stop]);
    this.background = toSrgb(options.background);
    this.intro = options.motion ? 0 : 1;

    this.resizeObserver = new ResizeObserver(this.handleResize);
    this.resizeObserver.observe(canvas);
    window.addEventListener("pointermove", this.handlePointerMove, { passive: true });
    canvas.addEventListener("webglcontextlost", this.handleContextLost);
    canvas.addEventListener("webglcontextrestored", this.handleContextRestored);

    this.resize();
    this.start();
  }

  set(next: Partial<GlyphFieldOptions>): void {
    const previous = this.options;
    this.options = { ...previous, ...next };

    if (next.background !== undefined) {
      this.background = toSrgb(next.background);
    }
    if (next.palette !== undefined) {
      this.paletteTarget = next.palette.map(toSrgb);
      if (!this.options.motion) {
        this.palette = this.paletteTarget.map((stop) => [...stop]);
      }
    }
    if (next.cell !== undefined && next.cell !== previous.cell) {
      this.buildAtlas();
    }
    if (next.motion !== undefined && next.motion !== previous.motion) {
      this.applyMotion();
      return;
    }
    if (!this.options.motion) {
      this.draw();
    }
  }

  /** A glyph shockwave from the form edge across the stage. */
  burst(): void {
    if (this.options.motion) {
      this.pulse = 0;
    }
  }

  /** Widens the ribbon after a successful sign-in; eased under motion. */
  setDone(done: boolean): void {
    this.doneTarget = done ? 1 : 0;
    if (!this.options.motion) {
      this.done = this.doneTarget;
      this.draw();
    }
  }

  pause(): void {
    this.running = false;
    cancelAnimationFrame(this.frame);
  }

  resume(): void {
    this.start();
  }

  destroy(): void {
    this.pause();
    this.resizeObserver.disconnect();
    window.removeEventListener("pointermove", this.handlePointerMove);
    this.canvas.removeEventListener("webglcontextlost", this.handleContextLost);
    this.canvas.removeEventListener("webglcontextrestored", this.handleContextRestored);
    if (this.program && !this.gl.isContextLost()) {
      this.gl.deleteTexture(this.program.texture);
      this.gl.deleteBuffer(this.program.buffer);
      this.gl.deleteProgram(this.program.program);
    }
    this.program = null;
  }

  private start(): void {
    if (!this.program) {
      return;
    }
    if (!this.options.motion) {
      this.draw();
      return;
    }
    if (this.running) {
      return;
    }
    this.running = true;
    this.last = performance.now();
    this.frame = requestAnimationFrame(this.tick);
  }

  private applyMotion(): void {
    if (this.options.motion) {
      this.start();
      return;
    }
    this.pause();
    this.intro = 1;
    this.pulse = 1;
    this.done = this.doneTarget;
    this.hover = 0;
    this.palette = this.paletteTarget.map((stop) => [...stop]);
    this.draw();
  }

  private readonly tick = (now: number): void => {
    if (!this.running) {
      return;
    }
    const dt = Math.min(MAX_FRAME_SECONDS, (now - this.last) / 1000);
    this.last = now;
    this.step(dt);
    this.draw();
    this.frame = requestAnimationFrame(this.tick);
  };

  private step(dt: number): void {
    this.time += dt * this.options.speed;
    this.intro = Math.min(1, this.intro + dt / INTRO_SECONDS);
    if (this.pulse < 1) {
      this.pulse = Math.min(1, this.pulse + dt / BURST_SECONDS);
    }
    this.done += (this.doneTarget - this.done) * Math.min(1, dt * 2.5);
    this.hover += (this.hoverTarget - this.hover) * Math.min(1, dt * 3);

    const follow = Math.min(1, dt * 6);
    this.smoothedMouse[0] += (this.mouse[0] - this.smoothedMouse[0]) * follow;
    this.smoothedMouse[1] += (this.mouse[1] - this.smoothedMouse[1]) * follow;

    const ease = Math.min(1, dt * 1.5);
    this.paletteTarget.forEach((target, index) => {
      const current = this.palette[index];
      for (let channel = 0; channel < 3; channel += 1) {
        current[channel] += (target[channel] - current[channel]) * ease;
      }
    });
  }

  private draw(): void {
    const program = this.program;
    if (!program || this.gl.isContextLost()) {
      return;
    }
    const { gl } = this;
    const { uniforms } = program;
    const motion = this.options.motion;

    gl.uniform2f(uniforms.uRes, this.canvas.width, this.canvas.height);
    gl.uniform1f(uniforms.uTime, this.time + TIME_OFFSET);
    gl.uniform2f(uniforms.uMouse, this.smoothedMouse[0], this.smoothedMouse[1]);
    gl.uniform1f(uniforms.uHover, motion ? this.hover : 0);
    gl.uniform1f(uniforms.uPulse, this.pulse);
    gl.uniform1f(uniforms.uDone, this.done);
    gl.uniform1f(uniforms.uIntro, this.intro);
    gl.uniform2f(uniforms.uCell, this.cellWidth, this.cellHeight);
    gl.uniform1i(uniforms.uAtlas, 0);
    gl.uniform1f(uniforms.uGlyphs, RAMP.length);
    gl.uniform3fv(uniforms.uBg, this.background);
    gl.uniform3fv(uniforms.uC0, this.palette[0]);
    gl.uniform3fv(uniforms.uC1, this.palette[1]);
    gl.uniform3fv(uniforms.uC2, this.palette[2]);
    gl.uniform3fv(uniforms.uC3, this.palette[3]);
    gl.uniform1f(uniforms.uDots, this.options.dots);
    gl.drawArrays(gl.TRIANGLES, 0, 3);
  }

  private buildAtlas(): void {
    const program = this.program;
    if (!program) {
      return;
    }
    const { gl, atlas } = this;
    const height = Math.max(1, Math.round(this.options.cell * this.dpr));
    const width = Math.max(1, Math.round(this.options.cell * CELL_WIDTH_RATIO * this.dpr));
    this.cellWidth = width;
    this.cellHeight = height;

    atlas.width = width * RAMP.length;
    atlas.height = height;
    const context = atlas.getContext("2d");
    if (!context) {
      return;
    }
    context.clearRect(0, 0, atlas.width, atlas.height);
    context.fillStyle = "white";
    context.textAlign = "center";
    context.textBaseline = "middle";
    context.font = `500 ${Math.round(height * GLYPH_SIZE_RATIO)}px ${this.options.fontFamily}`;
    RAMP_GLYPHS.forEach((glyph, index) => {
      context.fillText(glyph, index * width + width / 2, height * 0.54);
    });

    gl.activeTexture(gl.TEXTURE0);
    gl.bindTexture(gl.TEXTURE_2D, program.texture);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, atlas);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
  }

  private resize(): void {
    if (!this.program) {
      return;
    }
    this.dpr = Math.min(window.devicePixelRatio || 1, MAX_DPR);
    const rect = this.canvas.getBoundingClientRect();
    this.canvas.width = Math.max(1, Math.round(rect.width * this.dpr));
    this.canvas.height = Math.max(1, Math.round(rect.height * this.dpr));
    this.gl.viewport(0, 0, this.canvas.width, this.canvas.height);
    this.buildAtlas();
  }

  private readonly handleResize = (): void => {
    this.resize();
    if (!this.options.motion) {
      this.draw();
    }
  };

  private readonly handlePointerMove = (event: PointerEvent): void => {
    const rect = this.canvas.getBoundingClientRect();
    this.mouse = [(event.clientX - rect.left) * this.dpr, (rect.bottom - event.clientY) * this.dpr];
    this.hoverTarget =
      event.clientX >= rect.left &&
      event.clientX <= rect.right &&
      event.clientY >= rect.top &&
      event.clientY <= rect.bottom
        ? 1
        : 0;
    if (this.smoothedMouse[0] <= OFFSCREEN) {
      this.smoothedMouse = [this.mouse[0], this.mouse[1]];
    }
  };

  private readonly handleContextLost = (event: Event): void => {
    event.preventDefault();
    this.pause();
    this.program = null;
  };

  private readonly handleContextRestored = (): void => {
    this.program = buildProgram(this.gl);
    this.resize();
    this.start();
  };
}
