"use client"

import { motion } from "framer-motion"

const ICON_PATHS: Record<string, string> = {
  RESTAURANT:
    "M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z",
  SOCIAL_LIFE:
    "M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z",
  EVENT:
    "M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM7 10h5v5H7z",
}

export function PostPinSVG({
  color,
  category,
  isClicked,
}: {
  color: string
  category: string
  isClicked?: boolean
}) {
  const path = ICON_PATHS[category] || ICON_PATHS.RESTAURANT

  return (
    <motion.div
      className="relative group cursor-pointer"
      animate={
        isClicked
          ? {
              scale: [1, 1.35, 0.88, 1.06, 1],
              rotate: [0, -6, 4, -2, 0],
            }
          : { scale: 1, rotate: 0 }
      }
      transition={{
        duration: 0.5,
        ease: [0.34, 1.56, 0.64, 1],
        times: [0, 0.2, 0.45, 0.7, 1],
      }}
    >
      {isClicked && (
        <motion.span
          className="absolute inset-0 rounded-full"
          initial={{ scale: 0.6, opacity: 0.6 }}
          animate={{ scale: 2.5, opacity: 0 }}
          transition={{ duration: 0.6, ease: "easeOut" }}
          style={{ backgroundColor: color, filter: "blur(2px)" }}
        />
      )}

      <div className="relative">
        <svg
          width="28"
          height="36"
          viewBox="0 0 28 36"
          className="drop-shadow-lg"
        >
          <path
            d="M14 0C6.27 0 0 6.27 0 14c0 7.72 14 22 14 22s14-14.28 14-22C28 6.27 21.73 0 14 0z"
            fill={color}
            stroke="rgba(255,255,255,0.5)"
            strokeWidth="1"
          />
          <circle cx="14" cy="14" r="5" fill="white" opacity="0.9" />
          <path
            d={path}
            fill={color}
            transform="translate(9.5, 9.5) scale(0.75)"
          />
        </svg>
      </div>
    </motion.div>
  )
}
