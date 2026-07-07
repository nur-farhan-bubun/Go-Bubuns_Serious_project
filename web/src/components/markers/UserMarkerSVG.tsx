"use client"

export function UserMarkerSVG({
  color,
  pulse,
  size = 36,
}: {
  color: string
  pulse?: boolean
  size?: number
}) {
  return (
    <div className="relative" style={{ width: size, height: size }}>
      {pulse && (
        <div
          className="absolute -inset-2 rounded-full animate-ping opacity-40"
          style={{ backgroundColor: color }}
        />
      )}
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill={color}
        stroke="#fff"
        strokeWidth="1.5"
        width={size}
        height={size}
        className="drop-shadow-lg"
      >
        <path d="M12 2C7.6 2 4 5.6 4 10c0 5.5 7.3 11.5 7.6 11.7.2.2.6.2.8 0C12.7 21.5 20 15.5 20 10c0-4.4-3.6-8-8-8zm0 11a3 3 0 1 1 0-6 3 3 0 0 1 0 6z" />
      </svg>
    </div>
  )
}
