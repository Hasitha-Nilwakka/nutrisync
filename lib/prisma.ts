import { PrismaClient } from '../src/generated/prisma/client';
import { PrismaPg } from '@prisma/adapter-pg';
import {Pool} from 'pg'

// 1. Initialize the PostgreSQL connection pool using pg
const connectionString = process.env.DATABASE_URL;
const pool = new Pool({ connectionString });
const adapter = new PrismaPg(pool);

// 2. Setup the globalThis object type for Prisma preservation in development
const globalForPrisma = globalThis as unknown as {
  prisma: PrismaClient | undefined;
};

// 3. Instantiate or reuse the client instance
export const prisma =
  globalForPrisma.prisma ?? new PrismaClient({ adapter });

// 4. Cache the instance on globalThis in non-production environments
if (process.env.NODE_ENV !== 'production') {
  globalForPrisma.prisma = prisma;
}

export default prisma;
