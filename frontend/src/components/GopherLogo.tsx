import type { FC } from 'react';

interface GopherLogoProps {
  size?: number;
}

const GopherLogo: FC<GopherLogoProps> = ({ size = 32 }) => (
  <svg width={size} height={size} viewBox="0 0 40 40">
    <defs>
      <linearGradient id="glg" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stopColor="#4f6ef7" />
        <stop offset="100%" stopColor="#8b5cf6" />
      </linearGradient>
    </defs>
    <rect width="40" height="40" rx="10" fill="url(#glg)" />
    <ellipse cx="20" cy="22" rx="10" ry="9" fill="#e8d5b7" />
    <circle cx="15" cy="17" r="4.5" fill="#e8d5b7" />
    <circle cx="25" cy="17" r="4.5" fill="#e8d5b7" />
    <circle cx="15" cy="17" r="3" fill="white" />
    <circle cx="25" cy="17" r="3" fill="white" />
    <circle cx="15.5" cy="17.2" r="1.5" fill="#1a1a2e" />
    <circle cx="25.5" cy="17.2" r="1.5" fill="#1a1a2e" />
    <ellipse cx="20" cy="24" rx="3.5" ry="2" fill="#c9a67a" />
    <rect x="18.5" y="22" width="1.4" height="2.5" rx="0.7" fill="#b8935a" />
    <rect x="20.2" y="22" width="1.4" height="2.5" rx="0.7" fill="#b8935a" />
    <ellipse cx="14.5" cy="19.5" rx="1.5" ry="0.8" fill="#f4a0a0" />
    <ellipse cx="25.5" cy="19.5" rx="1.5" ry="0.8" fill="#f4a0a0" />
    <g transform="translate(26,26)">
      <rect x="0" y="3" width="8" height="6" rx="1" fill="white" opacity="0.9" />
      <path
        d="M2 3V2a2 2 0 0 1 4 0v1"
        stroke="white"
        strokeWidth="1.2"
        fill="none"
        opacity="0.9"
      />
    </g>
  </svg>
);

export default GopherLogo;
