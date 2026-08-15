import Image from "next/image";

type GroupLogoProps = {
  groupId: string;
  hasLogo: boolean;
  name: string;
  className?: string;
  imgClassName?: string;
};

export function GroupLogo({
  groupId,
  hasLogo,
  name,
  className = "h-10 w-10 rounded-xl bg-green-50 text-base font-bold text-green-700",
  imgClassName = "h-10 w-10 rounded-xl object-cover",
}: GroupLogoProps) {
  if (!hasLogo) {
    return (
      <span
        className={`flex shrink-0 items-center justify-center ${className}`}
        aria-hidden="true"
      >
        {name.charAt(0).toUpperCase()}
      </span>
    );
  }

  return (
    <Image
      src={`/api/v1/groups/${groupId}/logo`}
      alt=""
      width={40}
      height={40}
      unoptimized
      className={`shrink-0 ${imgClassName}`}
    />
  );
}