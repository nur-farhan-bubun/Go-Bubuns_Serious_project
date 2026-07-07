"use client"

import { motion } from "framer-motion"
import { UserMarkerSVG } from "./UserMarkerSVG"

export function HapticUserMarkerSVG({
  color,
  pulse,
  isClicked,
}: {
  color: string
  pulse?: boolean
  isClicked?: boolean
}) {
  return (
    <motion.div
      className="relative"
      animate={
        isClicked
          ? {
              scale: [1, 1.35, 0.88, 1.06, 1],
              rotate: [0, -6, 4, -2, 0],
            }
          : pulse
            ? { scale: [1, 1.08, 1] }
            : { scale: 1 }
      }
      transition={
        isClicked
          ? { duration: 0.5, ease: [0.34, 1.56, 0.64, 1], times: [0, 0.2, 0.45, 0.7, 1] }
          : pulse
            ? { duration: 0.8, repeat: Infinity, ease: "easeInOut" }
            : { duration: 0.2 }
      }
    >
      {isClicked && (
        <motion.span
          className="absolute -inset-1 rounded-full"
          initial={{ scale: 0.6, opacity: 0.5 }}
          animate={{ scale: 2.5, opacity: 0 }}
          transition={{ duration: 0.6, ease: "easeOut" }}
          style={{ backgroundColor: color, filter: "blur(2px)" }}
        />
      )}
      <UserMarkerSVG color={color} pulse={pulse} />
    </motion.div>
  )
}
