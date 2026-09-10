/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useId, type SVGProps } from 'react'

// Official Agnes AI mark (black circle, white "A" glyph + dot) sourced from
// platform.agnes-ai.com /images/biglogo.svg.
type IconAgnesProps = SVGProps<SVGSVGElement> & {
  size?: number
}

export function IconAgnes({ size = 20, ...props }: IconAgnesProps) {
  const uid = useId()
  const clipId = `agnes-clip-${uid}`
  const maskId = `agnes-mask0-${uid}`
  const dotMaskId = `agnes-mask1-${uid}`

  return (
    <svg
      xmlns='http://www.w3.org/2000/svg'
      viewBox='0 0 284 284'
      fill='none'
      width={size}
      height={size}
      {...props}
    >
      <g clipPath={`url(#${clipId})`}>
        <path
          d='M141.65 283.3C219.881 283.3 283.3 219.881 283.3 141.65C283.3 63.4189 219.881 0 141.65 0C63.4189 0 0 63.4189 0 141.65C0 219.881 63.4189 283.3 141.65 283.3Z'
          fill='#000000'
        />
        <path
          d='M42.67 168.24C42.67 168.24 25.78 220.34 56.91 225.59C88.04 230.84 139.65 215.87 196.7 106.67C196.7 106.67 189.66 140.96 185.83 160.61C184.67 166.59 179.44 170.92 173.35 170.97L165.74 171.02C163.5 171.04 162.65 173.9 164.49 175.18C171.7 180.17 180.95 190.98 182.43 213.89C182.75 218.78 187.89 221.92 192.36 219.9C194.36 219 196.3 217.44 197.79 214.82C199.04 212.63 199.62 210.11 199.72 207.59L198.69 198.04C198.14 192.98 201.17 188.27 205.95 186.54C212.71 184.09 221.84 178.96 226.31 168.44C226.84 167.2 225.61 165.91 224.34 166.37C219.82 167.99 211.23 170.67 203.76 170.63C201.54 170.62 199.86 168.6 200.18 166.4L210.55 93.9502C211.15 89.7802 207.99 86.0002 203.78 85.9102C201.44 85.8602 198.61 86.1502 195.36 87.1202C188.09 89.2902 181.21 92.9602 177.44 100.2C159.64 134.44 74.4 237.07 42.67 168.24Z'
          fill='#ffffff'
        />
        <mask
          id={maskId}
          style={{ maskType: 'luminance' }}
          maskUnits='userSpaceOnUse'
          x='38'
          y='85'
          width='189'
          height='142'
        >
          <path
            d='M42.67 168.24C42.67 168.24 25.78 220.34 56.91 225.59C88.04 230.84 139.65 215.87 196.7 106.67C196.7 106.67 189.66 140.96 185.83 160.61C184.67 166.59 179.44 170.92 173.35 170.97L165.74 171.02C163.5 171.04 162.65 173.9 164.49 175.18C171.7 180.17 180.95 190.98 182.43 213.89C182.75 218.78 187.89 221.92 192.36 219.9C194.36 219 196.3 217.44 197.79 214.82C199.04 212.63 199.62 210.11 199.72 207.59L198.69 198.04C198.14 192.98 201.17 188.27 205.95 186.54C212.71 184.09 221.84 178.96 226.31 168.44C226.84 167.2 225.61 165.91 224.34 166.37C219.82 167.99 211.23 170.67 203.76 170.63C201.54 170.62 199.86 168.6 200.18 166.4L210.55 93.9502C211.15 89.7802 207.99 86.0002 203.78 85.9102C201.44 85.8602 198.61 86.1502 195.36 87.1202C188.09 89.2902 181.21 92.9602 177.44 100.2C159.64 134.44 74.4 237.07 42.67 168.24Z'
            fill='#ffffff'
          />
        </mask>
        <g mask={`url(#${maskId})`}>
          <path d='M521.88 11.2002H-5.25V293.47H521.88V11.2002Z' fill='#ffffff' />
        </g>
        <path
          d='M197.34 71.25C207.016 71.25 214.86 63.406 214.86 53.73C214.86 44.0539 207.016 36.21 197.34 36.21C187.664 36.21 179.82 44.0539 179.82 53.73C179.82 63.406 187.664 71.25 197.34 71.25Z'
          fill='#ffffff'
        />
        <mask
          id={dotMaskId}
          style={{ maskType: 'luminance' }}
          maskUnits='userSpaceOnUse'
          x='179'
          y='36'
          width='36'
          height='36'
        >
          <path
            d='M197.34 71.25C207.016 71.25 214.86 63.406 214.86 53.73C214.86 44.0539 207.016 36.21 197.34 36.21C187.664 36.21 179.82 44.0539 179.82 53.73C179.82 63.406 187.664 71.25 197.34 71.25Z'
            fill='#ffffff'
          />
        </mask>
        <g mask={`url(#${dotMaskId})`}>
          <path d='M521.88 11.2002H-5.25V293.47H521.88V11.2002Z' fill='#ffffff' />
        </g>
      </g>
      <defs>
        <clipPath id={clipId}>
          <rect width='283.31' height='283.31' fill='#ffffff' />
        </clipPath>
      </defs>
    </svg>
  )
}
