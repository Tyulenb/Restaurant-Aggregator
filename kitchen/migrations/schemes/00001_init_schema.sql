-- +goose Up

CREATE TABLE "establishment" (
    "id" uuid NOT NULL,
    "name" text NOT NULL,
    "description" text,
    "available" boolean NOT NULL,
    "created_at" timestamp with time zone NOT NULL,
    "updated_at" timestamp with time zone NOT NULL,

    PRIMARY KEY ("id")
);

CREATE TABLE "order_status" (
    "id" integer NOT NULL,
    "status" text NOT NULL UNIQUE,

    PRIMARY KEY ("id")
);

CREATE TABLE "inventory_type" (
    "id" integer NOT NULL,
    "inventory_type" text NOT NULL UNIQUE,

    PRIMARY KEY ("id")
);

CREATE TABLE "category" (
    "id" uuid NOT NULL,
    "establishment_id" uuid NOT NULL,
    "external_id" text NOT NULL,
    "name" text NOT NULL,
    "description" text,
    "created_at" timestamp with time zone NOT NULL,
    "updated_at" timestamp with time zone NOT NULL,
    "deleted_at" timestamp with time zone,

    PRIMARY KEY ("id"),
    CONSTRAINT "uq_category_id_establishment_id" UNIQUE ("id", "establishment_id"),
    CONSTRAINT "uq_external_id_establishment_id" UNIQUE ("external_id", "establishment_id"),
    CONSTRAINT "fk_category_establishment_id_establishment_id"
        FOREIGN KEY ("establishment_id") REFERENCES "establishment"("id")
);

CREATE TABLE "product" (
    "id" uuid NOT NULL,
    "external_id" text NOT NULL,
    "establishment_id" uuid NOT NULL,
    "category_id" uuid NOT NULL,
    "name" text NOT NULL,
    "description" text,
    "available" boolean NOT NULL,
    "inventory_type_id" integer NOT NULL,
    "quantity" integer,
    "price" numeric NOT NULL,
    "created_at" timestamp with time zone NOT NULL,
    "updated_at" timestamp with time zone NOT NULL,
    "deleted_at" timestamp with time zone,

    PRIMARY KEY ("id"),
    CONSTRAINT "uq_external_id_establishment_id" UNIQUE ("external_id", "establishment_id"),
    CONSTRAINT "chk_product_quantity_nonnegative"
        CHECK ("quantity" IS NULL OR "quantity" >= 0),
    CONSTRAINT "chk_product_price_nonnegative"
        CHECK ("price" >= 0),
    CONSTRAINT "fk_product_category_establishment"
        FOREIGN KEY ("category_id", "establishment_id")
        REFERENCES "category"("id", "establishment_id"),
    CONSTRAINT "fk_product_establishment_id_establishment_id"
        FOREIGN KEY ("establishment_id") REFERENCES "establishment"("id"),
    CONSTRAINT "fk_product_inventory_type_id_inventory_type_id"
        FOREIGN KEY ("inventory_type_id") REFERENCES "inventory_type"("id")
);

CREATE TABLE "orders" (
    "id" uuid NOT NULL,
    "idempotency_key" uuid NOT NULL UNIQUE,
    "establishment_id" uuid NOT NULL,
    "customer_id" uuid NOT NULL,
    "delivery_id" uuid NOT NULL,
    "status_id" integer NOT NULL,
    "total_price" numeric NOT NULL,
    "rejection_reason" text,
    "created_at" timestamp with time zone NOT NULL,
    "updated_at" timestamp with time zone NOT NULL,

    PRIMARY KEY ("id"),
    CONSTRAINT "chk_orders_total_price_nonnegative"
        CHECK ("total_price" >= 0),
    CONSTRAINT "fk_orders_establishment_id_establishment_id"
        FOREIGN KEY ("establishment_id") REFERENCES "establishment"("id"),
    CONSTRAINT "fk_orders_status_id_order_status_id"
        FOREIGN KEY ("status_id") REFERENCES "order_status"("id")
);

CREATE TABLE "order_item" (
    "id" uuid NOT NULL,
    "order_id" uuid NOT NULL,
    "product_id" uuid NOT NULL,
    "external_product_id" text NOT NULL,
    "product_name" text NOT NULL,
    "unit_price" numeric NOT NULL,
    "quantity" integer NOT NULL,
    "total_sum" numeric NOT NULL,

    PRIMARY KEY ("id"),
    CONSTRAINT "chk_order_item_unit_price_nonnegative"
        CHECK ("unit_price" >= 0),
    CONSTRAINT "chk_order_item_quantity_positive"
        CHECK ("quantity" > 0),
    CONSTRAINT "chk_order_item_total_sum_nonnegative"
        CHECK ("total_sum" >= 0),
    CONSTRAINT "fk_order_item_order_id_orders_id"
        FOREIGN KEY ("order_id") REFERENCES "orders"("id"),
    CONSTRAINT "fk_order_item_product_id_product_id"
        FOREIGN KEY ("product_id") REFERENCES "product"("id")
);

-- External IDs are unique only inside an establishment and only for active rows.
CREATE UNIQUE INDEX "uq_category_establishment_external_id_active"
    ON "category" ("establishment_id", "external_id")
    WHERE "deleted_at" IS NULL;

CREATE UNIQUE INDEX "uq_product_establishment_external_id_active"
    ON "product" ("establishment_id", "external_id")
    WHERE "deleted_at" IS NULL;

-- Index foreign-key columns used for joins and parent-row checks/deletes.
CREATE INDEX "idx_category_establishment_id"
    ON "category" ("establishment_id");

CREATE INDEX "idx_product_establishment_id"
    ON "product" ("establishment_id");

CREATE INDEX "idx_product_category_establishment_id"
    ON "product" ("category_id", "establishment_id");

CREATE INDEX "idx_product_inventory_type_id"
    ON "product" ("inventory_type_id");

CREATE INDEX "idx_orders_establishment_id"
    ON "orders" ("establishment_id");

CREATE INDEX "idx_orders_status_id"
    ON "orders" ("status_id");

CREATE INDEX "idx_order_item_order_id"
    ON "order_item" ("order_id");

CREATE INDEX "idx_order_item_product_id"
    ON "order_item" ("product_id");

-- +goose Down

DROP TABLE IF EXISTS "order_item";
DROP TABLE IF EXISTS "orders";
DROP TABLE IF EXISTS "product";
DROP TABLE IF EXISTS "category";
DROP TABLE IF EXISTS "inventory_type";
DROP TABLE IF EXISTS "order_status";
DROP TABLE IF EXISTS "establishment";
