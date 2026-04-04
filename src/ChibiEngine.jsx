import React, { useMemo } from 'react';

// --- Grid ---------------------------------------------------------------------
const SCALE = 3;

// --- Palette (True Gaara Colors) -----------------------------------------------
const C = {
  O: '#000000', // Outline / Eyeliner
  H: '#8f2022', // Hair Dark Red (True Gaara Hair)
  h: '#691819', // Hair Shadow / Deep Spikes
  S: '#fceddd', // Skin (Very pale)
  s: '#e4cbba', // Skin shadow
  R: '#b42728', // Kanji Red
  E: '#c1e5d4', // Pale Mint Eyes (True Eye Color)
  e: '#000000', // Eye dark pupil
  W: '#ffffff', // Eye sclera
  M: '#000000', // Mouth
  P: '#5c524b', // Straps
  p: '#3b342e', // Straps shadow
  C: '#651c20', // Coat Dark Red
  c: '#4a1417', // Coat shadow
  T: '#d5c4a7', // Gourd light
  t: '#ab9573', // Gourd dark
  g: '#2d2b31', // Pants
  o: '#fceddd', // Hands
  _: 'transparent',
  X: 'transparent'
};

function buildKitsune(mood, isBlinking) {
  const px = [];

  // True Anime Spiky Hair & Accurate Face Proportions
  let RAW_FRAME = [
    "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
    "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
    "XXXXXXXXXXXXhhhhhhhhhhXXXXXXXXXXXXXXXXXX",
    "XXXXXXXXXhhhhHHHHHHHHHhhhhXXXXXXXXXXXXXX",
    "XXXXXXXhhHHHHHHHHHHHHHHHHHhhXXXXXXXXXXXX",
    "XXXXXhhHHHHHHHHHHHHHHHHHHHHHhhXXXXXXXXXX",
    "XXXXhHHhHHHHHHHHHHHHHHHHHHHHHHhXXXXXXXXX",
    "XXXhHHhHHHHHHHHHHHHHHHHHHHHHHHHhXXXXXXXX",
    "XXXhHHhHHHHHHHHHHHHHHHHhHHRRHHHhXXXXXXXX",
    "XXhHHhHHHHHHHHHHHHHHHHHhSShRHHHhXXXXXXXX",
    "XXhHhHHHHHHHHHHHHHHHHHHhSSSShHHhXXXXXXXX",
    "XXhHhHHHHHHHHHHHHHHHHHhHSSSSshHhHHhXXXXX",
    "XXhHHhHHHHHHHHHHHHHHHhHSSSSSSssHHHhXXXXX",
    "XXXhHHhHHHHHHHHHHHHHhHSSSSSSSsHShHHhXXXX",
    "XXXhHhHHHHHHHHHHHHhHhhSSSSSSSsSOHHhXXXXX",
    "XXXXhOHHHHHHHHHHhhhSSSSSSSSSSsSOHHhXXXXX",
    "XXXXXOSOOOOOOOOOSSSSSOOOOOOOOOSSOXXXXXXX", // 16 (Thick Eyeliner completely wraps eye)
    "XXXXXOSOWWWWWWeOOSSSSOWWWWWWeOOSOXXXXXXX", // 17 (White Sclera + Dark pupil)
    "XXXXXOSOWWEEEeeOOSSSSOWWEEEeeOOSOXXXXXXX", // 18 (Mint Iris + Dark Pupil)
    "XXXXXOSOOOOOOOOOSSSSSOOOOOOOOOSSOXXXXXXX", // 19 (Thick Bottom Eyeliner)
    "XXXXXhSSSSSssssSSSSSSSSSSssssSSSOXXXXXXX", // 20
    "XXXXXHOSSSSSSSSSSMMMMSSSSSSSSSSSOXXXXXXX", // 21 (Zero nose! Just a straight flat mouth)
    "XXXXXOHOSSSSSSSSSSSSSSSSSSSSSSSOXXXXXXXX", // 22
    "XXXXXXHOSSSSSSSSSSSSSSSSSSSSSSOXXXXXXXXX", // 23
    "XXXXXXXOOSSSSSSSSSSSSSSSSSSSOOXXXXXXXXXX", // 24
    "XXXXXXXXXOOOOOOOOOOOOOOOOOOOXXXXXXXXXXXX", // 25
    "XXXXXXXXXXOOPPPPCCCCOOSSOOXXXXXXTTTTOXXX", // 26
    "XXXXXXXXOOCoOPPCCCCPPOOOOOoOXXOTTTTTTTOX", // 27
    "XXXXXXXXOOCCPPPPCOOOoooooOOOXXOTTTTPPppX", // 28
    "XXXXXXXXXOCCOPCCCPCOOOOOOOXXXOPPTTTTttTX", // 29
    "XXXXXXXXXXXOOCCOOOOPCCOXXXXXXOPPTTTTttTX", // 30
    "XXXXXXXXXXXOCCOCPCCCPCOXXXXXXOTTTTTTtttO", // 31
    "XXXXXXXXXXXOCCCCCCCCCCXOXXXXOTTTTTTttOOX", // 32
    "XXXXXXXXXXOOCCCCCCCCCCXOOXXXOTTTTttOOOXX", // 33
    "XXXXXXXXXOOCccccccccccCOOXXXOTTTTtOOXXXX", // 34
    "XXXXXXXXOOCcccOOOccccccCOOXXXOOTTtOOXXXX", // 35
    "XXXXXXXXOOcccOOOOOccccccCCOOXXOOOOXXXXXX", // 36
    "XXXXXXXXOOcOOOOXOOOOOccccCCOXXXXXXXXXXXX", // 37
    "XXXXXXXXXOOOXXXXXXXOOOOOOCOOXXXXXXXXXXXX", // 38
    "XXXXXXXXXXXXXXOOgggggOOOXXXXXXXXXXXXXXXX", // 39
    "XXXXXXXXXXXXXXOOgggggOOXXXXXXXXXXXXXXXXX", // 40
    "XXXXXXXXXXXXXXOOgggggOOXXXXXXXXXXXXXXXXX", // 41
    "XXXXXXXXXXXXXXOOOOOOOOOXXXXXXXXXXXXXXXXX"  // 42
  ];

  let frame = [...RAW_FRAME];

  if (mood === 'hyped') {
    frame[21] = "XXXXXHOSSSSSSSSOMMMMOOSSSSSSSSSOXXXXXXX";
    frame[22] = "XXXXXOHOSSSSSSSSOOOOOOSSSSSSSSSOXXXXXXXX";
  } else if (mood === 'exhausted') {
    frame[16] = "XXXXXOSSSSSSSSSSSSSSSSSSSSSSSSSSOXXXXXXX";
    frame[17] = "XXXXXOSOOOOOOOOOSSSSSOOOOOOOOOSSOXXXXXXX";
    frame[18] = "XXXXXOSOeeeeeeOOSSSSOSOeeeeeeOOSOXXXXXXX";
    frame[19] = "XXXXXOSOOOOOOOOOSSSSSOOOOOOOOOSSOXXXXXXX";
  }

  if (isBlinking && mood !== 'exhausted') {
    frame[16] = "XXXXXOSSSSSSSSSSSSSSSSSSSSSSSSSSOXXXXXXX";
    frame[17] = "XXXXXOSOOOOOOOOOSSSSSOOOOOOOOOSSOXXXXXXX";
    frame[18] = "XXXXXOSSSSSSSSSSSSSSSSSSSSSSSSSSOXXXXXXX";
    frame[19] = "XXXXXOSSSSSSSSSSSSSSSSSSSSSSSSSSOXXXXXXX";
  }

  for (let y = 0; y < frame.length; y++) {
    for (let x = 0; x < frame[y].length; x++) {
      let char = frame[y][x];
      if (C[char] && C[char] !== 'transparent') {
        px.push({
          x: (x + 3) * SCALE,
          y: (y + 5) * SCALE,
          width: SCALE,
          height: SCALE,
          fill: C[char]
        });
      }
    }
  }
  return px;
}

const MOOD_ANIM = {
  idle:      'animate-float',
  hyped:     'animate-bounce animate-[tiny-breathe_0.8s_ease-in-out_infinite]',
  exhausted: 'animate-pulse opacity-[0.8] grayscale-[30%] animate-[tiny-breathe_4s_ease-in-out_infinite]',
};

import { useState, useEffect } from 'react';

export default function ChibiEngine({ mood = 'idle', onContextMenu, onClick }) {
  const [isBlinking, setIsBlinking] = useState(false);

  useEffect(() => {
    // Generate a random blink interval
    const blinkLogic = () => {
      setIsBlinking(true);
      setTimeout(() => setIsBlinking(false), 150); // Blink duration 150ms
    };

    const interval = setInterval(() => {
      // 50% chance to blink to make it more organic
      if (Math.random() > 0.5) {
        blinkLogic();
        // Sometimes double blink
        if (Math.random() > 0.7) {
          setTimeout(blinkLogic, 200);
        }
      }
    }, 3000); // Check every 3 seconds

    return () => clearInterval(interval);
  }, []);

  const pixels = useMemo(() => buildKitsune(mood, isBlinking), [mood, isBlinking]);

  return (
    <div
      className={`relative w-full h-full flex items-end justify-center ${MOOD_ANIM[mood] || ''} cursor-pointer pointer-events-auto`}
      onContextMenu={onContextMenu}
      onClick={onClick}
    >
      <svg width="200" height="200" viewBox="0 0 200 200" className="drop-shadow-xl origin-bottom transition-transform duration-1000 ease-in-out hover:scale-105" shapeRendering="crispEdges">
        {pixels.map((p, i) => (
          <rect key={i} x={p.x} y={p.y} width={p.width} height={p.height} fill={p.fill} />
        ))}
      </svg>
    </div>
  );
}
