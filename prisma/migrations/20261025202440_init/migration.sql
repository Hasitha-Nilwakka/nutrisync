-- AlterTable
ALTER TABLE "User" ALTER COLUMN "verified" SET DEFAULT false,
ALTER COLUMN "verifiedAt" DROP NOT NULL,
ALTER COLUMN "verificationToken" DROP NOT NULL,
ALTER COLUMN "verificationTokenExpiredAt" DROP NOT NULL;
